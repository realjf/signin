package signin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/realjf/datetimeutil"
	"github.com/redis/go-redis/v9"
)

const (
	SignBit               int              = 1
	DefaultRedisKeyPrefix string           = "signin"
	DefaultSignInterval   time.Duration    = time.Duration(24) * time.Hour
	DefaultDateTimeFormat datetimeutil.DTF = datetimeutil.F_YYYYMMDDhhmmss_hyphen
)

type ISignIn interface {
	Sign(id string, date time.Time) (bool, error)                       // sign-in
	SignCount(id string, start, end int64) (int64, error)               // returns the number of sign-in days
	ConsecutiveSignCount(id string, startDate time.Time) (int64, error) // returns the number of consecutive sign-in days
	GetSignStates(id string, endDate time.Time) (map[string]int, error) // get the states of sign-in
	setRedisClient(*redis.Client) error
	setRedisCluster(*redis.ClusterClient) error
	setRedisKeyPrefix(prefix string)
	setSignInterval(d time.Duration) // sign-in interval
	setStartDate(startDate time.Time)
	setEndDate(endDate time.Time)
	setBitFieldType(bitType string)
	SetDebug(bool)
	Close() error
}

type signIn struct {
	client      *redis.Client
	cluster     *redis.ClusterClient
	debug       bool
	ctx         context.Context
	rkey_prefix string
	interval    time.Duration
	startDate   time.Time
	endDate     time.Time
	useEndDate  bool
	bitType     string
}

func NewSignIn(options ...Option) ISignIn {
	s := &signIn{
		debug:       false,
		rkey_prefix: DefaultRedisKeyPrefix,
		interval:    DefaultSignInterval,
		ctx:         context.Background(),
		bitType:     "u8",
		useEndDate:  false,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// signin
func (s *signIn) Sign(id string, date time.Time) (bool, error) {
	// get key
	key := s.newSignRedisKey(id)

	if date.After(time.Now().Add(s.interval)) {
		return false, fmt.Errorf("sign date must be before now")
	}

	offset, err := s.getOffset(date)
	if err != nil {
		return false, err
	}
	if s.debug {
		fmt.Printf("offset: %d\n", offset)
	}
	var cmd *redis.IntCmd
	if s.cluster != nil {
		// check if today is signed
		isSign := s.cluster.GetBit(s.ctx, key, offset)
		if isSign != nil && isSign.Err() != nil && isSign.Val() == 1 {
			// already signed
			return true, nil
		}
		if isSign.Err() != nil {
			return false, isSign.Err()
		}
		// signin
		cmd = s.cluster.SetBit(s.ctx, key, offset, SignBit)
	} else if s.client != nil {
		// check if today is signed
		isSign := s.client.GetBit(s.ctx, key, offset)
		if isSign != nil && isSign.Err() != nil && isSign.Val() == 1 {
			// already signed
			return true, nil
		}
		if isSign.Err() != nil {
			return false, isSign.Err()
		}
		// signin
		cmd = s.client.SetBit(s.ctx, key, offset, SignBit)
	} else {
		return false, fmt.Errorf("redis client invalid")
	}

	if cmd.Err() != nil {
		return false, cmd.Err()
	}

	return true, nil
}

func (s *signIn) getOffset(date time.Time) (int64, error) {
	if s.useEndDate && date.After(s.endDate) {
		date = s.endDate
	}
	return datetimeutil.GetPosFromF(
		DefaultDateTimeFormat,
		datetimeutil.ParseDateFromTime(DefaultDateTimeFormat, s.startDate),
		datetimeutil.ParseDateFromTime(DefaultDateTimeFormat, date),
		s.interval)
}

func (s *signIn) newSignRedisKey(id string) string {
	sdate := datetimeutil.ParseDateFromTime(datetimeutil.F_YYYYMMDDhhmmss, s.startDate)
	return fmt.Sprintf("%s:%s:%s", s.rkey_prefix, id, sdate)
}

// returns the number of days of sign-in
// start = 1, end = -1 means get all
func (s *signIn) SignCount(id string, start, end int64) (int64, error) {
	key := s.newSignRedisKey(id)
	bitcout := &redis.BitCount{
		Start: start,
		End:   end,
	}
	var cmd *redis.IntCmd
	if s.cluster != nil {
		cmd = s.cluster.BitCount(s.ctx, key, bitcout)
	} else if s.client != nil {
		cmd = s.client.BitCount(s.ctx, key, bitcout)
	} else {
		return 0, fmt.Errorf("redis client invalid")
	}
	if cmd == nil {
		return 0, fmt.Errorf("redis error: bitcount falied")
	}
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}

func (s *signIn) calcBitType(startDate time.Time, endDate time.Time) (string, error) {
	count, err := datetimeutil.CountFromTime(DefaultDateTimeFormat, startDate, endDate, s.interval)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("u%d", count), nil
}

// returns the number of days of consecutive sign-in
// start = 1, end = -1 means get all
func (s *signIn) ConsecutiveSignCount(id string, startDate time.Time) (int64, error) {
	key := s.newSignRedisKey(id)
	offset, err := s.getOffset(startDate)
	if err != nil {
		return 0, err
	}
	bitType, err := s.calcBitType(startDate, time.Now())
	if err != nil {
		return 0, err
	}
	if s.debug {
		fmt.Printf("bittype: %s\n", bitType)
	}
	args := []interface{}{
		"get",
		bitType,
		offset,
	}
	var cmd *redis.IntSliceCmd
	if s.cluster != nil {
		cmd = s.cluster.BitField(s.ctx, key, args...)
	} else if s.client != nil {
		cmd = s.client.BitField(s.ctx, key, args...)
	} else {
		return 0, fmt.Errorf("redis client invalid")
	}
	if cmd == nil {
		return 0, fmt.Errorf("redis error: bitfield falied")
	}
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	if s.debug {
		fmt.Printf("bitfield result length: %d\n", len(cmd.Val()))
	}
	if len(cmd.Val()) == 0 {
		return 0, fmt.Errorf("bitfield result length is 0")
	}

	signBitmap := cmd.Val()[0]
	if s.debug {
		fmt.Printf("bitmap value: %b\n", signBitmap)
	}
	var signedDays int64
	for {
		compareBitmap := signBitmap >> 1
		compareBitmap = compareBitmap << 1
		if compareBitmap == signBitmap {
			break
		} else {
			signedDays++
		}
		signBitmap = signBitmap >> 1
	}
	return signedDays, nil
}

// start = 1, end = -1 means get all
func (s *signIn) GetSignStates(id string, endDate time.Time) (map[string]int, error) {
	key := s.newSignRedisKey(id)
	offset, err := s.getOffset(endDate)
	if err != nil {
		return nil, err
	}
	bitType, err := s.calcBitType(s.startDate, endDate)
	if err != nil {
		return nil, err
	}
	if s.debug {
		fmt.Printf("bittype: %s\n", bitType)
	}
	args := []interface{}{
		"get",
		bitType,
		0,
	}
	var cmd *redis.IntSliceCmd
	if s.cluster != nil {
		cmd = s.cluster.BitField(s.ctx, key, args...)
	} else if s.client != nil {
		cmd = s.client.BitField(s.ctx, key, args...)
	} else {
		return nil, fmt.Errorf("redis client invalid")
	}
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	if s.debug {
		fmt.Printf("bitfield result length: %d\n", len(cmd.Val()))
	}
	if len(cmd.Val()) == 0 || cmd.Val()[0] == 0 {
		return nil, nil
	}
	signBitmap := cmd.Val()[0]
	if s.debug {
		fmt.Printf("bitmap value: %b\n", signBitmap)
	}
	count, err := datetimeutil.CountFromTime(DefaultDateTimeFormat, s.startDate, endDate, s.interval)
	if err != nil {
		return nil, err
	}
	states := make(map[string]int)
	for i := count; i > 0; i-- {
		if i > offset {
			continue
		}

		datetime, err := datetimeutil.AddDuration(
			DefaultDateTimeFormat,
			datetimeutil.ParseDateFromTime(DefaultDateTimeFormat, s.startDate),
			time.Duration(i-1)*s.interval,
		)
		if err != nil {
			return nil, err
		}
		compareBitmap := signBitmap >> 1
		compareBitmap = compareBitmap << 1
		if compareBitmap != signBitmap {
			states[datetime] = 1
		} else {
			states[datetime] = 0
		}
		signBitmap = signBitmap >> 1
	}

	return states, nil
}

func (s *signIn) setRedisClient(c *redis.Client) error {
	if c == nil {
		return errors.New("invalid redis client")
	}
	s.client = c

	if s.debug {
		status := s.client.Ping(context.Background())
		if status == nil {
			return errors.New("failed to ping the client")
		}
		if status.Err() != nil {
			return status.Err()
		}
		fmt.Printf("%s\n", status.String())
	}

	return nil
}

func (s *signIn) setRedisCluster(c *redis.ClusterClient) error {
	if c == nil {
		return errors.New("invalid redis cluster client")
	}
	s.cluster = c

	if s.debug {
		status := s.cluster.Ping(context.Background())
		if status == nil {
			return errors.New("failed to ping the cluster")
		}
		if status.Err() != nil {
			return status.Err()
		}
		fmt.Printf("%s\n", status.String())
	}
	return nil
}

func (s *signIn) Close() error {
	if s.cluster != nil {
		return s.cluster.Close()
	}
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

func (s *signIn) SetDebug(debug bool) {
	s.debug = debug
}

func (s *signIn) setRedisKeyPrefix(prefix string) {
	s.rkey_prefix = prefix
}

func (s *signIn) setSignInterval(d time.Duration) {
	s.interval = d
}

func (s *signIn) setStartDate(startDate time.Time) {
	s.startDate = startDate
}

func (s *signIn) setEndDate(endDate time.Time) {
	s.endDate = endDate
	s.useEndDate = true
}

func (s *signIn) setBitFieldType(bitType string) {
	s.bitType = bitType
}
