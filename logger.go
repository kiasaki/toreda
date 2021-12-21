package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TimeSource interface {
	Now() time.Time
}

type Logger interface {
	LogDebug(component, message string, vals ...interface{})
	LogInfo(component, message string, vals ...interface{})
	LogWarn(component, message string, vals ...interface{})
	LogError(component, message string, vals ...interface{})
}

type FileLogger struct {
	timeSource  TimeSource
	baseFolder  string
	environment string
	truncate    bool
	fileHandles map[string]*os.File
}

func NewFileLogger(timeSource TimeSource, baseFolder, environment string, truncate bool) *FileLogger {
	return &FileLogger{
		timeSource:  timeSource,
		baseFolder:  baseFolder,
		environment: environment,
		truncate:    truncate,
		fileHandles: map[string]*os.File{},
	}
}

func (l *FileLogger) Log(level, component, message string, vals ...interface{}) {
	dateString := l.timeSource.Now().Format("2006-01-02")
	dateTimeString := l.timeSource.Now().Format("2006-01-02 15:04:05")
	path := filepath.Join(l.baseFolder, l.environment, dateString, "log.txt")
	if l.environment != "production" {
		path = filepath.Join(l.baseFolder, l.environment, "log.txt")
	}
	fileHandle, err := l.fileHandleFor(path)
	if err != nil {
		fmt.Println("file_logger: " + err.Error())
		return
	}
	log := fmt.Sprintf(
		"%s|%s|%s|%s| %s\n",
		dateTimeString,
		l.environment,
		level,
		component,
		fmt.Sprintf(message, vals...),
	)
	if level == "error" {
		fmt.Print(log)
	}
	_, err = fileHandle.Write([]byte(log))
	if err != nil {
		fmt.Println("file_logger: " + err.Error())
		return
	}
}

func (l *FileLogger) LogDebug(component, message string, vals ...interface{}) {
	l.Log("debug", component, message, vals...)
}

func (l *FileLogger) LogInfo(component, message string, vals ...interface{}) {
	l.Log("info", component, message, vals...)
}

func (l *FileLogger) LogWarn(component, message string, vals ...interface{}) {
	l.Log("warn", component, message, vals...)
}

func (l *FileLogger) LogError(component, message string, vals ...interface{}) {
	l.Log("error", component, message, vals...)
}

func (l *FileLogger) fileHandleFor(path string) (*os.File, error) {
	var ok bool
	var err error
	var file *os.File
	if file, ok = l.fileHandles[path]; !ok {
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, errors.New("error creating directory " + path)
		}
		openMode := os.O_WRONLY | os.O_CREATE
		if l.truncate {
			openMode = openMode | os.O_TRUNC
		} else {
			openMode = openMode | os.O_APPEND
		}
		if file, err = os.OpenFile(path, openMode, 0666); err != nil {
			return nil, errors.New("error opening log file " + path + " (" + err.Error() + ")")
		}
		l.fileHandles[path] = file
	}
	return file, nil
}

type ConsoleLogger struct {
}

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}

func (l *ConsoleLogger) Log(level, component, message string, vals ...interface{}) {
	dateTimeString := time.Now().In(timeLocation).Format("2006-01-02 15:04:05")
	fmt.Printf(
		"%s|%s|%s|%s| %s\n",
		dateTimeString,
		"console",
		level,
		component,
		fmt.Sprintf(message, vals...),
	)
}

func (l *ConsoleLogger) LogDebug(component, message string, vals ...interface{}) {
	l.Log("debug", component, message, vals...)
}

func (l *ConsoleLogger) LogInfo(component, message string, vals ...interface{}) {
	l.Log("info", component, message, vals...)
}

func (l *ConsoleLogger) LogWarn(component, message string, vals ...interface{}) {
	l.Log("warn", component, message, vals...)
}

func (l *ConsoleLogger) LogError(component, message string, vals ...interface{}) {
	l.Log("error", component, message, vals...)
}
