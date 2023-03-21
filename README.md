# signin

sign-in data structure on Redis cache (redis中的签到数据结构)

### required

- redis >= v3.2.0

### features

- count the number of consecutive sign-in days (统计连续签到天数)
- count the number of sign-in days (统计签到天数)
- sign in (进行签到)
- get sign-in states (获取签到状态)

### Q&A

#### Question 1: What is the number of consecutive sign-in days?(问题1：什么叫做连续签到天数？)

Counting forward from the last check-in until the first non-sign-in is encountered, the total number of check-ins is calculated, which is the number of consecutive sign-in days (从最后一次签到开始向前统计，直到遇到第一次未签到为止，计算总的签到次数，就是连续签到天数)
