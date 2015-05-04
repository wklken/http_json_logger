// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// 这份代码是修改beego/logs模块, 删除封装精简后的结果
// this file was modified from beego/logs, under apache license 2.0
// source code url:https://github.com/astaxie/beego/tree/master/logs

package logs

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// ========== MuxWriter ==========

// an *os.File writer with locker.
type MuxWriter struct {
	sync.Mutex
	fd *os.File
}

// write to os.File.
func (l *MuxWriter) Write(b []byte) (int, error) {
	l.Lock()
	defer l.Unlock()
	return l.fd.Write(b)
}

// set os.File in writer.
func (l *MuxWriter) SetFd(fd *os.File) {
	if l.fd != nil {
		l.fd.Close()
	}
	l.fd = fd
}

// ========== FileLogWriter =========

// It writes messages by lines limit, file size limit, or time frequency.
type FileLogWriter struct {
	*log.Logger
	mw *MuxWriter

	// The opened file
	Filename string `json:"filename"`
	// Rotate daily
	Daily          bool `json:"daily"`
	daily_opendate int

	Rotate bool `json:"rotate"`

	startLock sync.Mutex // Only one log can write to the file
}

// Init file logger with json config.
// jsonconfig like:
//	{
//	"filename":"logs/beego.log",
//	"daily":true,
//	"rotate":true
//	}
func (w *FileLogWriter) Init(jsonconfig string) error {
	err := json.Unmarshal([]byte(jsonconfig), w)
	if err != nil {
		return err
	}
	if len(w.Filename) == 0 {
		return errors.New("jsonconfig must have filename")
	}
	err = w.initLogger()
	return err
}

// start file logger. create log file and set to locker-inside file writer.
func (w *FileLogWriter) initLogger() error {
	fd, err := w.createLogFile()
	if err != nil {
		return err
	}
	w.mw.SetFd(fd)
	return w.initFd()
}

func (w *FileLogWriter) createLogFile() (*os.File, error) {
	// Open the log file
	fd, err := os.OpenFile(w.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	return fd, err
}

func (w *FileLogWriter) initFd() error {
	w.daily_opendate = time.Now().Day()
	// w.daily_opendate = 1
	return nil
}

func (w *FileLogWriter) docheck() {
	w.startLock.Lock()
	defer w.startLock.Unlock()

	if w.Rotate && (w.Daily && time.Now().Day() != w.daily_opendate) {
		if err := w.DoRotate(); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
			return
		}
	}
}

// write logger message into file.
func (w *FileLogWriter) WriteMsg(msg string) error {
	w.docheck()
	w.Logger.Println(msg)
	return nil
}

// DoRotate means it need to write file in new file.
// new file name like xx.log.2013-01-01.2
func (w *FileLogWriter) DoRotate() error {
	_, err := os.Lstat(w.Filename)
	if err != nil {
		return nil
	}

	fname := ""

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour).Format("20060102")

	fname = w.Filename + fmt.Sprintf(".%s", yesterday)

	_, err = os.Lstat(fname)
	if err == nil {
		num := 1
		for ; err == nil && num <= 999; num++ {
			fname = w.Filename + fmt.Sprintf(".%s.%03d", yesterday, num)
			_, err = os.Lstat(fname)
		}
	}

	// block Logger's io.Writer
	w.mw.Lock()
	defer w.mw.Unlock()

	fd := w.mw.fd
	fd.Close()

	// close fd before rename
	// Rename the file to its newfound home
	err = os.Rename(w.Filename, fname)
	if err != nil {
		return fmt.Errorf("Rotate: %s\n", err)
	}

	// re-start logger
	err = w.initLogger()
	if err != nil {
		return fmt.Errorf("Rotate StartLogger: %s\n", err)
	}

	return nil
}

// flush file logger.
// there are no buffering messages in file logger in memory.
// flush file means sync file from disk.
func (w *FileLogWriter) Flush() {
	w.mw.fd.Sync()
}

// destroy file logger, close file writer.
func (w *FileLogWriter) Destroy() {
	w.mw.fd.Close()
}

// ========= JsonLogger =========

type JsonLogger struct {
	msg    chan string
	writer FileLogWriter
}

func NewLogger(channellen int64, config string) *JsonLogger {
	jl := new(JsonLogger)
	jl.msg = make(chan string, channellen)

	// init writer
	w := FileLogWriter{
		Filename: "",
		Daily:    true,
		Rotate:   true,
	}
	// use MuxWriter instead direct use os.File for lock write when rotate
	// set MuxWriter as Logger's io.Writer
	w.mw = new(MuxWriter)
	// without any format
	w.Logger = log.New(w.mw, "", 0)

	err := w.Init(config)
	jl.writer = w
	if err != nil {
		fmt.Println("logs.BeeLogger.SetLogger: " + err.Error())
		return nil
	}
	go jl.startLogger()
	return jl
}

func (j *JsonLogger) startLogger() {
	for {
		select {
		case msg := <-j.msg:
			err := j.writer.WriteMsg(msg)
			if err != nil {
				fmt.Println("ERROR, unable to WriteMsg:", err)
			}
		}
	}
}

func (j *JsonLogger) WriteJson(v ...interface{}) {
	msg := fmt.Sprintf("%s", v...)
	j.msg <- msg
}

func (j *JsonLogger) Close() {
	for {
		if len(j.msg) > 0 {
			msg := <-j.msg
			err := j.writer.WriteMsg(msg)
			if err != nil {
				fmt.Println("ERROR, unable to WriteMsg (while closing logger):", err)
			}
			continue
		}
		break
	}
	j.writer.Flush()
	j.writer.Destroy()
}

// ===================================
