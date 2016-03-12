// * This file contains an implementation of a
// * [Logrus Formatter](https://github.com/Sirupsen/logrus#formatters) according to VMware CNA
// * storage team specifications. The `appendKeyValue` and `needsQuoting` functions are copied
// * from the implementation of the TextFormatter in Logrus.

package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"strings"
)

type VmwareFormatter struct{}

func (f *VmwareFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	fmt.Fprint(b, entry.Time.String())
	b.WriteByte(' ')
	b.WriteByte('[')
	fmt.Fprint(b, strings.ToUpper(entry.Level.String()))
	b.WriteByte(']')
	b.WriteByte(' ')
	b.WriteString(entry.Message)
	for key, value := range entry.Data {
		f.appendKeyValue(b, key, value)
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

func (f *VmwareFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {

	b.WriteString(key)
	b.WriteByte('=')

	switch value := value.(type) {
	case string:
		if needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	case error:
		errmsg := value.Error()
		if needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	default:
		fmt.Fprint(b, value)
	}

	b.WriteByte(' ')
}
