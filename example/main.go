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
		cluster   bool
		startdate string
	}{
		addr:      "redis-12996.c60.us-west-1-2.ec2.cloud.redislabs.com:12996",
		password:  "jK8YanVRZr2W7yr0cIQTLRbyyaH7twco",
		cluster:   false,
		startdate: "20230301",
		username:  "default",
	}
	sdate, err := datetimeutil.ParseDate2Time(datetimeutil.F_YYYYMMDD, ts.startdate)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	s := signin.NewSignIn(
		signin.WithRedisClient(ts.addr, ts.password, ts.username),
		signin.WithSignInterval(time.Duration(1)*time.Second),
		signin.WithStartDate(sdate),
		signin.WithDebug(),
		// signin.WithBitFieldType("u64"),
	)
	defer s.Close()

	for i := int64(0); i < 10; i++ {
		if rand.Intn(2) == 1 {
			date := sdate.Add(time.Duration(i) * time.Second)
			s.Sign("1", date)
		}

		time.Sleep(1 * time.Second)
	}
	total, err := s.SignCount("1", 0, 9)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	fmt.Printf("total: %d\n", total)
	total1, err := s.ConsecutiveSignCount("1", sdate)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	fmt.Printf("total: %d\n", total1)
	states, err := s.GetSignStates("1", time.Now())
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	fmt.Printf("states: %v\n", states)
}
