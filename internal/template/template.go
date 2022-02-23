package template

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/mylxsw/coll"
)

func Render(content string, wr io.Writer, data interface{}) error {
	funcMap := template.FuncMap{
		"have":    haveValue,
		"haveAll": haveAllValue,
		"haveAny": haveAnyValue,
		"default": defaultValue,
		"indent":  indentLeft,
	}

	temp, err := template.New("index.html").Funcs(funcMap).Parse(content)
	if err != nil {
		return err
	}

	return temp.Execute(wr, data)
}

func haveAllValue(vals ...interface{}) bool {
	for _, val := range vals {
		if !haveValue(val) {
			return false
		}
	}

	return true
}

func haveAnyValue(vals ...interface{}) bool {
	for _, val := range vals {
		if haveValue(val) {
			return true
		}
	}

	return false
}

func haveValue(val interface{}) bool {
	return val != nil
}

func defaultValue(val interface{}, defaultVal interface{}) interface{} {
	if val != nil {
		return val
	}

	return defaultVal
}

func indentLeft(ident string, message string) string {
	result := coll.MustNew(strings.Split(message, "\n")).Map(func(line string) string {
		return ident + line
	}).Reduce(func(carry string, line string) string {
		return fmt.Sprintf("%s\n%s", carry, line)
	}, "").(string)

	return strings.Trim(result, "\n")
}
