package main

import (
	"strings"
	"time"
)

type (
	Cron struct {
		jobs map[string](chan bool)
	}
)

func newCron() *Cron {
	log.Info("Init Cron")
	return &Cron{
		jobs: make(map[string]chan bool),
	}
}

func (cron *Cron) normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (cron *Cron) New(name string, f func(), duration time.Duration) {
	name = cron.normalizeName(name)
	log.Info("New cron: ", name)

	if name == "" {
		log.Fatal("Empty name for cronjob provided")
	}

	ticker := time.NewTicker(duration)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				log.Info("Cron ", name, " function called")
				f()

			case <-done:
				return
			}
		}
	}()

	cron.jobs[name] = done
}

func (cron *Cron) Stop(name string) {
	name = cron.normalizeName(name)
	if done, ok := cron.jobs[name]; ok {
		log.Info("Stopping cron: ", name)
		done <- true
		delete(cron.jobs, name)
	}
}
