package config

import (
	"fmt"
	"github.com/loeffel-io/ls-lint/v2/internal/rule"
	"reflect"
	"strings"
	"sync"
)

type Ls map[string]interface{}
type RuleIndex map[string]map[string][]rule.Rule

const (
	sep = string('/')
	or  = " | "
)

type Config struct {
	Ls     Ls       `yaml:"ls"`
	Ignore []string `yaml:"ignore"`
	*sync.RWMutex
}

func NewConfig(ls Ls, ignore []string) *Config {
	return &Config{
		Ls:      ls,
		Ignore:  ignore,
		RWMutex: new(sync.RWMutex),
	}
}

func (config *Config) GetLs() Ls {
	config.RLock()
	defer config.RUnlock()

	return config.Ls
}

func (config *Config) GetIgnore() []string {
	config.RLock()
	defer config.RUnlock()

	return config.Ignore
}

func (config *Config) GetIgnoreIndex() map[string]bool {
	var ignoreIndex = make(map[string]bool)

	for _, path := range config.GetIgnore() {
		ignoreIndex[path] = true
	}

	return ignoreIndex
}

func (config *Config) ShouldIgnore(ignoreIndex map[string]bool, path string) bool {
	if ignore, exists := ignoreIndex[path]; exists {
		return ignore
	}

	dirs := strings.Split(path, sep)
	for i := 0; i < len(dirs); i++ {
		if ignore, exists := ignoreIndex[strings.Join(dirs[:i], sep)]; exists {
			return ignore
		}
	}

	return false
}

func (config *Config) GetConfig(index RuleIndex, path string) map[string][]rule.Rule {
	dirs := strings.Split(path, sep)

	for i := len(dirs); i >= 0; i-- {
		if find, exists := index[strings.Join(dirs[:i], sep)]; exists {
			return find
		}
	}

	return nil
}

func (config *Config) GetIndex(list Ls) (RuleIndex, error) {
	var index = make(RuleIndex)

	if err := config.walkIndex(index, "", list); err != nil {
		return nil, err
	}

	return index, nil
}

func (config *Config) walkIndex(index RuleIndex, key string, list Ls) error {
	if index[key] == nil {
		index[key] = make(map[string][]rule.Rule)
	}

	for k, v := range list {
		if v == nil {
			continue
		}

		if reflect.TypeOf(v).Kind() == reflect.Map {
			switch key == "" {
			case true:
				if err := config.walkIndex(index, k, v.(Ls)); err != nil {
					return err
				}
			case false:
				var keyCombination = fmt.Sprintf("%s%s%s", key, sep, k)
				if err := config.walkIndex(index, keyCombination, v.(Ls)); err != nil {
					return err
				}
			}

			continue
		}

		for _, ruleName := range strings.Split(v.(string), or) {
			ruleName = strings.TrimSpace(ruleName)
			ruleSplit := strings.SplitN(ruleName, ":", 2)
			ruleName = ruleSplit[0]

			if r, exists := rule.Rules[ruleName]; exists {
				r = config.copyRule(r)

				if err := r.SetParameters(ruleSplit[1:]); err != nil {
					return fmt.Errorf("rule %s failed with %s", ruleName, err.Error())
				}

				index[key][k] = append(index[key][k], r)
				continue
			}

			return fmt.Errorf("rule %s not exists", ruleName)
		}
	}

	return nil
}

func (config *Config) copyRule(r rule.Rule) rule.Rule {
	switch r.GetName() {
	case "regex":
		return new(rule.Regex).Init()
	case "disallow":
		return new(rule.Disallow).Init()
	}

	return r
}
