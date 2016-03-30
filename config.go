package rpc

import (
	"github.com/xplacepro/common"
	"io/ioutil"
)

func ParseConfiguration(path string, conf *map[string]string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	*conf = common.ParseValues(string(data), rune('='), '#')
}
