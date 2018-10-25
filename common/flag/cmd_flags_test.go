package flag

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"gopkg.in/urfave/cli.v1"
	"strconv"
	"testing"
)

var int64slice = cli.Int64Slice{100, 200, 300}
var (
	StringFlag = cli.StringFlag{
		Name:  "STRING",
		Value: "string",
	}
	StringNew = "new_string"

	IntFlag = cli.IntFlag{
		Name:  "INT",
		Value: -100,
	}
	IntNew = 100

	Int64Flag = cli.Int64Flag{
		Name:  "INT64",
		Value: -1800000000000,
	}
	Int64New = int64(1800000000000)

	Int64SliceFlag = cli.Int64SliceFlag{
		Name:  "Int64Slice",
		Value: &int64slice,
	}

	UIntFlag = cli.UintFlag{
		Name:  "UINT",
		Value: 2000000,
	}
	UIntNew = uint(300000)

	UInt64Flag = cli.Uint64Flag{
		Name:  "UInt64",
		Value: 5000000,
	}
	Uint64New = uint64(6000000)

	FloatFlag = cli.Float64Flag{
		Name:  "Float",
		Value: 2.147,
	}
	FloatNew = float64(3.156)

	BoolFlag = cli.BoolFlag{
		Name:   "Bool",
		Hidden: true,
	}
	BoolNew = false
)

func initFlags() []cli.Flag {
	flags := []cli.Flag{
		StringFlag,
		IntFlag,
		Int64Flag,
		Int64SliceFlag,
		UIntFlag,
		UInt64Flag,
		FloatFlag,
		BoolFlag,
	}

	return flags
}

func TestNewCmdFlags_Set(t *testing.T) {
	flagSet := new(flag.FlagSet)
	StringFlag.Apply(flagSet)
	IntFlag.Apply(flagSet)
	Int64Flag.Apply(flagSet)
	UIntFlag.Apply(flagSet)
	UInt64Flag.Apply(flagSet)
	FloatFlag.Apply(flagSet)
	BoolFlag.Apply(flagSet)

	context := cli.NewContext(nil, flagSet, nil)
	context.Set(StringFlag.Name, StringNew)
	context.Set(IntFlag.Name, strconv.FormatInt(int64(IntNew), 10))
	context.Set(Int64Flag.Name, strconv.FormatInt(Int64New, 10))
	context.Set(UIntFlag.Name, strconv.FormatUint(uint64(UIntNew), 10))
	context.Set(UInt64Flag.Name, strconv.FormatUint(Uint64New, 10))
	context.Set(FloatFlag.Name, strconv.FormatFloat(FloatNew, 'f', 3, 64))
	context.Set(BoolFlag.Name, strconv.FormatBool(BoolNew))

	flags := initFlags()
	cmdFlags := NewCmdFlags(context, flags)
	assert.NotNil(t, cmdFlags)

	isExist := cmdFlags.IsSet(StringFlag.Name)
	assert.Equal(t, true, isExist)

	strVal := cmdFlags.String(StringFlag.Name)
	assert.Equal(t, strVal, StringNew)

	isExist = cmdFlags.IsSet(IntFlag.Name)
	assert.Equal(t, true, isExist)
	intVal := cmdFlags.Int(IntFlag.Name)
	assert.Equal(t, intVal, IntNew)

	isExist = cmdFlags.IsSet(Int64Flag.Name)
	assert.Equal(t, true, isExist)
	int64Val := cmdFlags.Int64(Int64Flag.Name)
	assert.Equal(t, int64Val, Int64New)

	isExist = cmdFlags.IsSet(UIntFlag.Name)
	assert.Equal(t, true, isExist)
	uintVal := cmdFlags.Uint(UIntFlag.Name)
	assert.Equal(t, uintVal, UIntNew)

	isExist = cmdFlags.IsSet(UInt64Flag.Name)
	assert.Equal(t, true, isExist)
	uint64Val := cmdFlags.Uint64(UInt64Flag.Name)
	assert.Equal(t, uint64Val, Uint64New)

	isExist = cmdFlags.IsSet(FloatFlag.Name)
	assert.Equal(t, true, isExist)
	floatVal := cmdFlags.Float64(FloatFlag.Name)
	assert.Equal(t, floatVal, FloatNew)

	isExist = cmdFlags.IsSet(BoolFlag.Name)
	assert.Equal(t, true, isExist)
	boolVal := cmdFlags.Bool(BoolFlag.Name)
	assert.Equal(t, boolVal, BoolNew)
}

func TestNewCmdFlags(t *testing.T) {
	flagSet := new(flag.FlagSet)
	StringFlag.Apply(flagSet)
	IntFlag.Apply(flagSet)
	Int64Flag.Apply(flagSet)
	Int64SliceFlag.Apply(flagSet)
	UIntFlag.Apply(flagSet)
	UInt64Flag.Apply(flagSet)
	FloatFlag.Apply(flagSet)
	BoolFlag.Apply(flagSet)

	context := cli.NewContext(nil, flagSet, nil)

	flags := initFlags()
	cmdFlags := NewCmdFlags(context, flags)
	assert.NotNil(t, cmdFlags)

	isExist := cmdFlags.IsSet(StringFlag.Name)
	assert.Equal(t, false, isExist)
	strVal := cmdFlags.String(StringFlag.Name)
	assert.Equal(t, strVal, StringFlag.Value)

	isExist = cmdFlags.IsSet(IntFlag.Name)
	assert.Equal(t, false, isExist)
	intVal := cmdFlags.Int(IntFlag.Name)
	assert.Equal(t, intVal, IntFlag.Value)

	isExist = cmdFlags.IsSet(Int64Flag.Name)
	assert.Equal(t, false, isExist)
	int64Val := cmdFlags.Int64(Int64Flag.Name)
	assert.Equal(t, int64Val, Int64Flag.Value)

	isExist = cmdFlags.IsSet(UIntFlag.Name)
	assert.Equal(t, false, isExist)
	uintVal := cmdFlags.Uint(UIntFlag.Name)
	assert.Equal(t, uintVal, UIntFlag.Value)

	isExist = cmdFlags.IsSet(UInt64Flag.Name)
	assert.Equal(t, false, isExist)
	uint64Val := cmdFlags.Uint64(UInt64Flag.Name)
	assert.Equal(t, uint64Val, UInt64Flag.Value)

	isExist = cmdFlags.IsSet(FloatFlag.Name)
	assert.Equal(t, false, isExist)
	floatVal := cmdFlags.Float64(FloatFlag.Name)
	assert.Equal(t, floatVal, FloatFlag.Value)

	isExist = cmdFlags.IsSet(BoolFlag.Name)
	assert.Equal(t, false, isExist)
	boolVal := cmdFlags.Bool(BoolFlag.Name)
	assert.Equal(t, boolVal, false)
}

func TestCmdFlags_CheckExclusive(t *testing.T) {
	flagSet := new(flag.FlagSet)
	StringFlag.Apply(flagSet)
	IntFlag.Apply(flagSet)
	Int64Flag.Apply(flagSet)
	Int64SliceFlag.Apply(flagSet)
	UIntFlag.Apply(flagSet)
	UInt64Flag.Apply(flagSet)
	FloatFlag.Apply(flagSet)
	BoolFlag.Apply(flagSet)

	context := cli.NewContext(nil, flagSet, nil)
	context.Set(StringFlag.Name, StringNew)
	context.Set(IntFlag.Name, strconv.FormatInt(int64(IntNew), 10))
	context.Set(Int64Flag.Name, strconv.FormatInt(Int64New, 10))
	context.Set(UIntFlag.Name, strconv.FormatUint(uint64(UIntNew), 10))
	context.Set(UInt64Flag.Name, strconv.FormatUint(Uint64New, 10))

	flags := initFlags()
	cmdFlags := NewCmdFlags(context, flags)
	assert.NotNil(t, cmdFlags)

	assert.PanicsWithValue(t, "flags --STRING, --INT can't be used at the same time", func() {
		cmdFlags.CheckExclusive(StringFlag, IntFlag)
	})

	cmdFlags.CheckExclusive(FloatFlag, BoolFlag)
	cmdFlags.CheckExclusive(StringFlag, BoolFlag)
	cmdFlags.CheckExclusive(StringFlag, Int64SliceFlag)
}
