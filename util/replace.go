package util

import (
	"log"
	"regexp"
	"strings"

	"github.com/piggy-moe/subtitle/config"
)

type Replacer struct {
	rule  []config.Replace
	index []*regexp.Regexp
}

func MustNewReplacer(rule []config.Replace) Replacer {
	var err error
	index := make([]*regexp.Regexp, len(rule))
	for i, item := range rule {
		if !item.Regexp {
			continue
		}
		if index[i], err = regexp.Compile(item.From); err != nil {
			log.Fatalf("unable to compile regexp %s: %s", item.From, err.Error())
		}
	}
	return Replacer{rule, index}
}

func (r Replacer) Replace(s string) string {
	for i, item := range r.rule {
		if !item.Regexp {
			s = strings.ReplaceAll(s, item.From, item.To)
		} else {
			s = r.index[i].ReplaceAllString(s, item.To)
		}
	}
	return s
}
