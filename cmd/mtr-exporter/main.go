package main

// *mtr-exporter* periodically executes *mtr* to a given host and provides the
// measured results as prometheus metrics.

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"io/ioutil"
    "gopkg.in/yaml.v2"
	"github.com/robfig/cron/v3"
)
type Config struct {
	Hosts     []string  `yaml:"hosts"`
}


var config Config
func main() {

	log.SetFlags(0)

	configFile   := flag.String("config.file", "mtr.yaml", "MTR exporter configuration file.")
	rawTargets := flag.String("targets", "", "List of targets")
	mtrBin := flag.String("mtr", "mtr", "path to `mtr` binary")
	bind := flag.String("bind", ":8080", "bind address")
	schedule := flag.String("schedule", "@every 60s", "Schedule at which often `mtr` is launched")
	doPrintVersion := flag.Bool("version", false, "show version")
	doPrintUsage := flag.Bool("h", false, "show help")
	doTimeStampLogs := flag.Bool("tslogs", false, "use timestamps in logs")

	flag.Usage = usage
	flag.Parse()

	targets := strings.Split(*rawTargets, " ")
    
   

	if *doPrintVersion == true {
		printVersion()
		return
	}
	if *doPrintUsage == true {
		flag.Usage()
		return
	}
	if *doTimeStampLogs == true {
		log.SetFlags(log.LstdFlags | log.LUTC)
	}

	if len(targets) == 0  {
		log.Println("error: no mtr target given")
		os.Exit(1)
		return
	}

    yamlFile, err := ioutil.ReadFile(*configFile)

    if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Error parsing config file: %s", err)
	}
     
     j := len(config.Hosts)
     



	jobs := make([]*mtrJob, j)
	for i, target := range  config.Hosts {
		args := append([]string{target}, flag.Args()...)
		job := newMtrJob(*mtrBin, args)

		c := cron.New()
		_, _ = c.AddFunc(*schedule, func() {
			log.Println("launching", job.cmdLine)
			if err := job.Launch(); err != nil {
				log.Println("failed:", err)
				return
			}
			log.Println("done: ",
				len(job.Report.Hubs), "hops in", job.Duration, ".")
		})
		c.Start()
		jobs[i] = job
	}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        for _, job := range jobs {
            job.ServeHTTP(w, r)
		}
	})

	log.Println("serving /metrics at", *bind, "...")
	log.Fatal(http.ListenAndServe(*bind, nil))
}
