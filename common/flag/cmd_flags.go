package flag

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"strconv"
	"strings"
)

type flagInfo struct {
	IsSet bool
	Value string
}
type CmdFlags map[string]flagInfo

func NewCmdFlags(ctx *cli.Context, totalFlags []cli.Flag) CmdFlags {
	flags := make(CmdFlags, len(totalFlags))
	for _, f := range totalFlags {
		if ctx.GlobalIsSet(f.GetName()) {
			flags[f.GetName()] = flagInfo{true, ctx.GlobalString(f.GetName())}
		} else if ctx.IsSet(f.GetName()) {
			flags[f.GetName()] = flagInfo{true, ctx.String(f.GetName())}
		} else {
			// Get default flag value
			flags[f.GetName()] = flagInfo{false, ctx.String(f.GetName())}
		}
	}
	return flags
}

func (c CmdFlags) IsSet(name string) bool {
	info, ok := c[name]
	return ok && info.IsSet
}

// Bool returns false if not found
func (c CmdFlags) Bool(name string) bool {
	info, ok := c[name]
	if ok {
		if parsed, err := strconv.ParseBool(info.Value); err == nil {
			return parsed
		}
	}
	return false
}

// Float64 returns 0 if not found
func (c CmdFlags) Float64(name string) float64 {
	info, ok := c[name]
	if ok {
		if parsed, err := strconv.ParseFloat(info.Value, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// Int64 returns 0 if not found
func (c CmdFlags) Int64(name string) int64 {
	info, ok := c[name]
	if ok {
		if parsed, err := strconv.ParseInt(info.Value, 0, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// Int returns 0 if not found
func (c CmdFlags) Int(name string) int {
	info, ok := c[name]
	if ok {
		if parsed, err := strconv.ParseInt(info.Value, 0, 64); err == nil {
			return int(parsed)
		}
	}
	return 0
}

// Uint64 returns 0 if not found
func (c CmdFlags) Uint64(name string) uint64 {
	info, ok := c[name]
	if ok {
		if parsed, err := strconv.ParseUint(info.Value, 0, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// Uint returns 0 if not found
func (c CmdFlags) Uint(name string) uint {
	info, ok := c[name]
	if ok {
		if parsed, err := strconv.ParseUint(info.Value, 0, 64); err == nil {
			return uint(parsed)
		}
	}
	return 0
}

// String returns "" if not found
func (c CmdFlags) String(name string) string {
	info, ok := c[name]
	if ok {
		return info.Value
	}
	return ""
}

// checkExclusive verifies that only a single instance of the provided flags was set by the user.
func (c CmdFlags) CheckExclusive(args ...cli.Flag) {
	if len(args) <= 1 {
		return
	}

	set := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		name := args[i].GetName()
		if c.IsSet(name) {
			set = append(set, "--"+name)
		}
	}
	if len(set) > 1 {
		panic(fmt.Sprintf("flags %v can't be used at the same time", strings.Join(set, ", ")))
	}
}
