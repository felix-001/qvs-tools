package main

import "github.com/jart/gosip/sip"

func GetCallId(s string) (string, error) {
	msg, err := sip.ParseMsg([]byte(s))
	if err != nil {
		return "", err
	}
	return msg.CallID, nil
}
