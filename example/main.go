package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/realjf/datetimeutil"

	"github.com/realjf/signin"
)

func main() {
	ts := struct {
		addr      string
		addrs     []string
		password  string
		username  string
		startdate string
	}{
		addr:      "redis-12996.c60.us-west-1-2.ec2.cloud.redislabs.com:12996",
		password:  "jK8YanVRZr2W7yr0cIQTLRbyyaH7twco",
		startdate: "20230315",
		username:  "default",
	}
	sdate, err := datetimeutil.ParseDate2Time(datetimeutil.F_YYYYMMDD, ts.startdate)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	s := signin.NewSignIn(
		signin.WithRedisClient(ts.addr, ts.password, ts.username),
		signin.WithSignInterval(time.Duration(24)*time.Hour),
		signin.WithStartDate(sdate),
		signin.WithDebug(),
	)
	defer s.Close()

	for i := int64(0); i < 10; i++ {
		if rand.Intn(2) == 1 {
			date := sdate.Add(time.Duration(i*24) * time.Hour)
			ok, err := s.Sign("1", date)
			if !ok {
				fmt.Printf("sign [%s]: %s\n", datetimeutil.ParseDateFromTime(datetimeutil.F_YYYYMMDDhhmmss_hyphen, date), err.Error())
			} else {
				fmt.Printf("sign [%s]: success\n", datetimeutil.ParseDateFromTime(datetimeutil.F_YYYYMMDDhhmmss_hyphen, date))
			}
		}
	}
	total, err := s.SignCount("1", 0, -1)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	fmt.Printf("total sign: %d\n", total)
	total1, err := s.ConsecutiveSignCount("1", sdate)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	fmt.Printf("consecutive sign: %d\n", total1)
	states, err := s.GetSignStates("1", time.Now())
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	for d, st := range states {
		fmt.Printf("states: %s %d\n", d, st)
	}
}
