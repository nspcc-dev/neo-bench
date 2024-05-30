package internal

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var flags = pflag.NewFlagSet("cmd", pflag.ExitOnError)

func validateImport(in *string) error {
	if in == nil || *in == "" {
		return nil
	}

	file, err := os.Open(*in)
	if err != nil {
		return err
	}

	return file.Close()
}

// InitSettings returns settings based on flags and environment.
func InitSettings() *viper.Viper {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix("BENCH")
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// prints defaults and exit with passed code and messages
	exit := func(code int, msg ...string) {
		if code != 0 {
			msg[0] = color.New(color.FgRed).Sprint("\nERROR: ") + msg[0]
			msg = append(msg, "")
		}

		flags.PrintDefaults()

		for i := range msg {
			fmt.Println(msg[i])
		}

		os.Exit(code)
	}

	flags.SortFlags = false

	// `` at the beginning of help message allows to hide type of flag:

	help := flags.BoolP("help", "h", false, "Show usage message.")

	desc := flags.StringP("desc", "d", "unknown benchmark", "Benchmark description.")

	out := flags.StringP("out", "o", "report.log", "Path where report would be written.")

	mode := flags.StringP("mode", "m", ModeRate.String(),
		"``Benchmark mode.\n"+
			"Example: -m "+ModeWorker.String()+" --mode "+ModeRate.String())

	workers := flags.IntP("workers", "w", 30,
		"Number of used workers.\n"+
			"Example: -w 10 -w 15 -w 40")

	timeLimit := flags.DurationP("timeLimit", "z", time.Second*30,
		"The time limit when an application can send requests.\n"+
			"When the time limit is reached, application stops send requests and wait for parsing transactions.\n"+
			"Examples: -z 10s -z 3m")

	rateLimit := flags.IntP("rateLimit", "q", 1000, "QPS - queries per second, rate limit")

	concurrent := flags.IntP("concurrent", "c", 4,
		"Number of used cpu cores."+
			"Example: -c 4 --concurrent 8")

	rpcAddresses := flags.StringArrayP("rpcAddress", "a", []string{"127.0.0.1:20331"},
		"``RPC addresses for RPC calls to test nodes.\n"+
			"You can specify multiple addresses.\n"+
			"Example -a 127.0.0.1:80 -a 127.0.0.2:8080")

	reqTimeout := flags.DurationP("request_timeout", "t", DefaultTimeout,
		"Request timeout.\n"+
			"Used for RPC requests.\n"+
			"Example: -t 30s --request_timeout 15s")

	input := flags.StringP("in", "i", "",
		"``Path to input file to load transactions.\n"+
			"Example: -i ./dump.txs --in /path/to/import/transactions")

	flags.BoolP("vote", "", false, "Vote before the bench.")
	flags.BoolP("disable-stats", "", false, "Disable memory and CPU usage statistics collection.")

	if err := v.BindPFlags(flags); err != nil {
		panic(err)
	}

	if err := v.ReadConfig(DevNull); err != nil {
		panic(err)
	}

	if err := flags.Parse(os.Args); err != nil {
		panic(err)
	}

	if err := validateImport(input); err != nil {
		exit(2, err.Error())
	}

	switch {
	case help != nil && *help:
		exit(0)
	case reqTimeout == nil || *reqTimeout < 0:
		exit(2, "Request timeout could not be negative value.")
	case out == nil || *out == "":
		exit(2, "Report path could not be empty.")
	case desc == nil || *desc == "":
		exit(2, "Benchmark description could not be empty.")
	case mode == nil || *mode == "":
		exit(2, "Benchmark mode could not be empty.")
	case rpcAddresses == nil || len(*rpcAddresses) == 0:
		exit(2, "RPC addresses could not be empty.")
	case concurrent == nil || *concurrent <= 0:
		exit(2, "CPUs could not be empty or negative value.")
	case timeLimit == nil || *timeLimit <= 0:
		exit(2, "Time limit could not be empty or negative value.")
	}

	switch BenchMode(*mode) {
	case ModeWorker:
		switch {
		case workers == nil || *workers <= 0:
			exit(2, "Workers count could not be empty or negative value")
		}
	case ModeRate:
		switch {
		case rateLimit == nil || *rateLimit <= 0:
			exit(2, "Rate limit (QPS) could not be empty or negative value")
		}
	default:
		exit(2, "Unknown benchmark mode.")
	}

	// set RPC addresses (wrong parser in viper)
	v.Set("rpcAddress", *rpcAddresses)

	runtime.GOMAXPROCS(*concurrent)

	return v
}
