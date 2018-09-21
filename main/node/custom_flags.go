package node

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"github.com/LemoFoundationLtd/lemochain-go/common/math"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"os"
	"os/user"
	"path"
	"strings"
)

//
// type DirectoryString struct {
// 	Value string
// }
//
// func (ds *DirectoryString) String() string {
// 	return ds.Value
// }
//
// func (ds *DirectoryString) Set(value string) error {
// 	ds.Value = expandPath(value)
// 	return nil
// }
//
// func (ds *DirectoryString) Names() []string {
// 	return flagNames(ds)
// }
//
// type DirectoryFlag struct {
// 	Name  string
// 	Value DirectoryString
// 	Usage string
// }
//
// func (self DirectoryFlag) String() string {
// 	fmtString := "%s %v\t%v"
// 	if len(self.Value.Value) > 0 {
// 		fmtString = "%s \"%v\"\t%v"
// 	}
// 	return fmt.Sprintf(fmtString, prefixedNames(self.Name), self.Value.Value, self.Usage)
// }
//
// func (self DirectoryFlag) Apply(set *flag.FlagSet) {
// 	eachName(self.Name, func(name string) {
// 		set.Var(&self.Value, self.Name, self.Usage)
// 	})
// }

type TextMarshaler interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}

type textMarshalerVal struct {
	v TextMarshaler
}

func (v textMarshalerVal) String() string {
	if v.v == nil {
		return ""
	}
	text, _ := v.v.MarshalText()
	return string(text)
}

func (v textMarshalerVal) Set(s string) error {
	return v.v.UnmarshalText([]byte(s))
}

type TextMarshalerFlag struct {
	Name  string
	Value TextMarshaler
	Usage string
}

func (f TextMarshalerFlag) GetName() string {
	return f.Name
}

func (f TextMarshalerFlag) String() string {
	return fmt.Sprintf("%s \"%v\"\t%v", prefixedNames(f.Name), f.Value, f.Usage)
}

func (f TextMarshalerFlag) Apply(set *flag.FlagSet) {
	eachName(f.Name, func(name string) {
		set.Var(textMarshalerVal{f.Value}, f.Name, f.Usage)
	})
}

func GlobalTextMarshaler(ctx *cli.Context, name string) TextMarshaler {
	val := ctx.Generic(name)
	if val == nil {
		return nil
	}
	return val.(textMarshalerVal).v
}

type BigFlag struct {
	Name  string
	Value *big.Int
	Usage string
}

type bigValue big.Int

func (b *bigValue) String() string {
	if b == nil {
		return ""
	}
	return (*big.Int)(b).String()
}

func (b *bigValue) Set(s string) error {
	int, ok := math.ParseBig256(s)
	if !ok {
		return errors.New("invalid integer syntax")
	}
	*b = (bigValue)(*int)
	return nil
}

func (f BigFlag) GetName() string {
	return f.Name
}

func (f BigFlag) String() string {
	fmtString := "%s %v\t%v"
	if f.Value != nil {
		fmtString = "%s \"%v\"\t%v"
	}
	return fmt.Sprintf(fmtString, prefixedNames(f.Name), f.Value, f.Usage)
}

func (f BigFlag) Apply(set *flag.FlagSet) {
	eachName(f.Name, func(name string) {
		set.Var((*bigValue)(f.Value), f.Name, f.Usage)
	})
}

func GlobalBig(ctx *cli.Context, name string) *big.Int {
	val := ctx.Generic(name)
	if val == nil {
		return nil
	}
	return (*big.Int)(val.(*bigValue))
}

func eachName(longName string, fn func(string)) {
	parts := strings.Split(longName, ",")
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func prefixedNames(fullName string) (prefixed string) {
	parts := strings.Split(fullName, ",")
	for i, name := range parts {
		name = strings.Trim(name, " ")
		prefixed += prefixFor(name) + name
		if i < len(parts)-1 {
			prefixed += ", "
		}
	}
	return
}

func prefixFor(name string) (prefix string) {
	if len(name) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}
	return
}
