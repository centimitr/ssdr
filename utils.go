package ssdr

import (
	"fmt"
	"strings"
)

func log(vs ...interface{}) {
	fmt.Println(vs...)
}

func check(err error, prompts ...string) bool {
	if err != nil {
		prompt := strings.Join(prompts, ": ")
		if prompt != "" {
			prompt += ":"
			fmt.Println(prompt, err)
		}
		fmt.Println(err)
		return true
	}
	return false
}

type Handler func()

func (h *Handler) Handler() {
	if *h != nil {
		(*h)()
	}
}

type SubscribeHandler func(list ServiceListValue)

func (h *SubscribeHandler) Handle(list ServiceListValue) {
	if *h != nil {
		(*h)(list)
	}
}
