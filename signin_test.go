package signin_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/realjf/datetimeutil"

	"github.com/realjf/signin"
)

func TestSigninPing(t *testing.T) {
	cases := map[string]struct {
		addr      string
		addrs     []string
		password  string
		username  string
		cluster   bool
		startdate string
	}{
		"client": {
			addr:      "redis-12996.c60.us-west-1-2.ec2.cloud.redislabs.com:12996",
			password:  "jK8YanVRZr2W7yr0cIQTLRbyyaH7twco",
			cluster:   false,
			startdate: "20230301",
			username:  "default",
		},
		"cluster": {
			addr:      "",
			addrs:     []string{"redis-12996.c60.us-west-1-2.ec2.cloud.redislabs.com:12996"},
			password:  "jK8YanVRZr2W7yr0cIQTLRbyyaH7twco",
			cluster:   true,
			startdate: "20230301",
			username:  "default",
		},
	}

	for name, ts := range cases {
		t.Run(name, func(t *testing.T) {
			if !ts.cluster {
				sdate, err := datetimeutil.ParseDate2Time(datetimeutil.F_YYYYMMDD, ts.startdate)
				if err != nil {
					t.Errorf("%s", err)
					return
				}
				s := signin.NewSignIn(
					signin.WithRedisClient(ts.addr, ts.password, ts.username),
					signin.WithSignInterval(time.Duration(1)*time.Second),
					signin.WithStartDate(sdate),
					signin.WithDebug(),
					signin.WithBitFieldType("u63"),
				)
				defer s.Close()

				for i := int64(0); i < 10; i++ {
					if rand.Intn(2) == 1 {
						date := sdate.Add(time.Duration(i) * time.Second)
						t.Logf("date: %s", datetimeutil.ParseDateFromTime(datetimeutil.F_YYYYMMDDhhmmss_hyphen, date))
						ok, err := s.Sign("1", date)
						if !ok {
							t.Errorf("sign: %v", err)
						}
					}

					time.Sleep(1 * time.Second)
				}
				total, err := s.SignCount("1", 0, 9)
				if err != nil {
					t.Errorf("%s", err)
					return
				}
				t.Logf("total: %d", total)
				total1, err := s.ConsecutiveSignCount("1", sdate)
				if err != nil {
					t.Errorf("%s", err)
					return
				}
				t.Logf("total: %d", total1)
				// states, err := s.GetSignStates("1", 0, 9)
				// if err != nil {
				// 	t.Errorf("%s", err)
				// 	return
				// }
				// t.Logf("%v", states)
			} else {
				// s := signin.NewSignIn(signin.WithRedisCluster(ts.addrs, ts.password))
				// defer s.Close()
			}
		})
	}
}
