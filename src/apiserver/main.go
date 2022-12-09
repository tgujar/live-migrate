package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

type conf struct {
	NfsPath     string `yaml:"nfsPath"`
	ManagerURL  string `yaml:"managerURL"`
	ShellPath   string `yaml:"shellPath"`
	PreChkIters int    `yaml:"precheckpoint_iters"`
	PreChkDelay int    `yaml:"precheckpoint_delayms"`
}

func (c *conf) getConf(fname string) *conf {

	yamlFile, err := ioutil.ReadFile(fname)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func (c *conf) heartbeat() {
	for range time.Tick(time.Second * 10) {
		http.Get(c.ManagerURL + "/heartbeat")
	}
}

func (c *conf) Shellout(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(c.ShellPath, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (c *conf) getPath(id string, pre bool) string {
	fname := id + ".tar.gz"
	if pre {
		fname = "pre_" + fname
	}
	return filepath.Clean(c.NfsPath) + "/" + fname
}

func checkErr(errStr string, err error) error {
	if err != nil {
		log.Printf("Error: %v", err)
		return err
	} else if errStr != "" {
		log.Printf("Podman Error: %v", errStr)
		return fmt.Errorf(errStr)
	}
	return nil
}

func (c *conf) checkpoint(id string) error {

	for i := 0; i < c.PreChkIters; i++ {
		_, errStr, err := c.Shellout(fmt.Sprintf("podman container checkpoint -P -e %s %s", c.getPath(id, true), id))
		perr := checkErr(errStr, err)
		if perr != nil {
			return perr
		}
		time.Sleep(time.Duration(c.PreChkDelay * int(time.Millisecond)))
	}
	_, errStr, err := c.Shellout(fmt.Sprintf("podman container checkpoint --ignore-rootfs --file-locks --tcp-established --with-previous -e %s %s", c.getPath(id, false), id))
	if perr := checkErr(errStr, err); perr != nil {
		return perr
	}
	go c.Shellout(fmt.Sprintf("podman container rm -f %s", id))
	return nil
}

func (c *conf) CheckpointHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	err := c.checkpoint(id)
	resp := struct{ Error string }{Error: ""}
	if err != nil {
		resp.Error = fmt.Sprint(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (c *conf) restore(id string) error {
	_, errStr, err := c.Shellout(fmt.Sprintf("podman container restore --ignore-rootfs --file-locks --tcp-established --import-previous %s --import %s", c.getPath(id, true), c.getPath(id, false)))
	perr := checkErr(errStr, err)
	return perr
}

func (c *conf) RestoreHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	err := c.restore(id)
	resp := struct{ Error string }{Error: ""}
	if err != nil {
		resp.Error = fmt.Sprint(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (c *conf) create(image string) error {
	_, errStr, err := c.Shellout(fmt.Sprintf("podman run -d %s", image))
	perr := checkErr(errStr, err)
	return perr
}

func (c *conf) CreateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("image")
	err := c.create(id)
	resp := struct{ Error string }{Error: ""}
	if err != nil {
		resp.Error = fmt.Sprint(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	var c conf
	c.getConf("params.yaml")
	http.HandleFunc("/checkpoint", c.CheckpointHandler)
	http.HandleFunc("/restore", c.RestoreHandler)
	http.HandleFunc("/create", c.CreateHandler)
	go c.heartbeat()
	http.ListenAndServe("0.0.0.0:8080", nil)
}
