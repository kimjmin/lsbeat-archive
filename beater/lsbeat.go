package beater

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/kimjmin/lsbeat/config"
)

type Lsbeat struct {
	beatConfig   *config.Config
	done         chan struct{}
	period       time.Duration
	client       publisher.Client
	path         string    //root 디렉토리
	lasIndexTime time.Time //가장 마지막에 검색한 시간
}

// Creates beater
func New() *Lsbeat {
	return &Lsbeat{
		done: make(chan struct{}),
	}
}

/// *** Beater interface methods ***///

func (bt *Lsbeat) Config(b *beat.Beat) error {

	// Load beater beatConfig
	err := b.RawConfig.Unpack(&bt.beatConfig)
	if err != nil {
		return fmt.Errorf("Error reading config file: %v", err)
	}

	return nil
}

func (bt *Lsbeat) Setup(b *beat.Beat) error {

	// Setting default period if not set
	if bt.beatConfig.Lsbeat.Period == "" {
		bt.beatConfig.Lsbeat.Period = "1s"
	}

	// Setting default Path if not set.
	if bt.beatConfig.Lsbeat.Path == "" {
		bt.beatConfig.Lsbeat.Path = "."
	}
	bt.path = bt.beatConfig.Lsbeat.Path

	bt.client = b.Publisher.Connect()

	var err error
	bt.period, err = time.ParseDuration(bt.beatConfig.Lsbeat.Period)
	if err != nil {
		return err
	}

	return nil
}

func (bt *Lsbeat) Run(b *beat.Beat) error {
	logp.Info("lsbeat is running! Hit CTRL-C to stop it.")

	ticker := time.NewTicker(bt.period)
	counter := 1
	bt.lasIndexTime = time.Now()
	for {
		listDir(bt.path, bt, b, counter)
		bt.lasIndexTime = time.Now()
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}
		// event := common.MapStr{
		// 	"@timestamp": common.Time(time.Now()),
		// 	"type":       b.Name,
		// 	"counter":    counter,
		// }
		// bt.client.PublishEvent(event)

		logp.Info("Event sent")
		counter++
	}
}

func (bt *Lsbeat) Cleanup(b *beat.Beat) error {
	return nil
}

func (bt *Lsbeat) Stop() {
	close(bt.done)
}

//scan dirs and files
func listDir(dirFile string, bt *Lsbeat, b *beat.Beat, counter int) {
	files, _ := ioutil.ReadDir(dirFile)
	for _, f := range files {
		t := f.ModTime()
		//fmt.Printf("%T\n", t)
		// fmt.Println(f.Name(), dirFile+"/"+f.Name(), f.IsDir(), t, f.Size())

		event := common.MapStr{
			"@timestamp":  common.Time(time.Now()),
			"type":        b.Name,
			"counter":     counter,
			"modTime":     t,
			"filename":    f.Name(),
			"fullname":    dirFile + "/" + f.Name(),
			"isDirectory": f.IsDir(),
			"fileSize":    f.Size(),
		}
		if counter == 1 {
			bt.client.PublishEvent(event)
		} else {
			if t.After(bt.lasIndexTime) {
				bt.client.PublishEvent(event)
			}
		}

		if f.IsDir() {
			listDir(dirFile+"/"+f.Name(), bt, b, counter)
		}
	}
}
