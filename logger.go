package applogger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Logger it loads the config for logging
type Logger struct {
	// DisableColor default behavior is to log with color
	DisableColor bool
	// DataTimeUTC default behavior is to log at UTC
	DataTimeUTC bool
}

const (
	// LevelDebug logs everything
	LevelDebug int32 = 1

	// LevelInfo logs Info, Warnings and Errors
	LevelInfo int32 = 2

	// LevelWarn logs Warning and Errors
	LevelWarn int32 = 4

	// LevelError logs just Errors
	LevelError int32 = 8
)

// for coloring the std
const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorCyan

	colorBold     = 1
	colorDarkGray = 90
)

// ApplicationLog provides support to write to log files.
type ApplicationLog struct {
	LogLevel int32
	Debug    *log.Logger
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	File     *log.Logger
	LogFile  *os.File
}

// log maintains a pointer to a singleton for the logging system.
var logger ApplicationLog

// Start initializes ApplicationLog and only displays the specified logging level.
func (l *Logger) Start(logLevel int32) {
	l.turnOnLogging(logLevel, nil)
}

// StartFile initializes tracelog and only displays the specified logging level
// and creates a file to capture writes.
func (l *Logger) StartFile(logLevel int32, baseFilePath string, daysToKeep int) {
	baseFilePath = strings.TrimRight(baseFilePath, "/")
	currentDate := time.Now().UTC()
	dateDirectory := time.Now().UTC().Format("2006-01-02")
	dateFile := currentDate.Format("2006-01-02T15-04-05")

	filePath := fmt.Sprintf("%s/%s/", baseFilePath, dateDirectory)
	fileName := strings.Replace(fmt.Sprintf("%s.txt", dateFile), " ", "-", -1)

	err := os.MkdirAll(filePath, os.ModePerm)
	if err != nil {
		log.Fatalf("main : Start : Failed to Create log directory : %s : %s\n", filePath, err)
	}

	logf, err := os.Create(fmt.Sprintf("%s%s", filePath, fileName))
	if err != nil {
		log.Fatalf("main : Start : Failed to Create log file : %s : %s\n", fileName, err)
	}

	// Turn the logging on
	l.turnOnLogging(logLevel, logf)

	// Cleanup any existing directories
	l.LogDirectoryCleanup(baseFilePath, daysToKeep)
}

// Stop will release resources and shutdown all processing.
func (l *Logger) Stop() error {
	l.Started("Stop")

	var err error
	if logger.LogFile != nil {
		Debug("Stop", "Closing File")
		err = logger.LogFile.Close()
	}

	l.Completed("Stop")
	return err
}

// LogLevel returns the configured logging level.
func LogLevel() int32 {
	return atomic.LoadInt32(&logger.LogLevel)
}

// turnOnLogging configures the logging writers.
func (l *Logger) turnOnLogging(logLevel int32, fileHandle io.Writer) {
	debugHandle := ioutil.Discard
	infoHandle := ioutil.Discard
	warnHandle := ioutil.Discard
	errorHandle := ioutil.Discard

	if logLevel&LevelDebug != 0 {
		debugHandle = os.Stdout
		infoHandle = os.Stdout
		warnHandle = os.Stdout
		errorHandle = os.Stderr
	}

	if logLevel&LevelInfo != 0 {
		infoHandle = os.Stdout
		warnHandle = os.Stdout
		errorHandle = os.Stderr
	}

	if logLevel&LevelWarn != 0 {
		warnHandle = os.Stdout
		errorHandle = os.Stderr
	}

	if logLevel&LevelError != 0 {
		errorHandle = os.Stderr
	}

	if fileHandle != nil {
		if debugHandle == os.Stdout {
			debugHandle = io.MultiWriter(fileHandle, debugHandle)
		}

		if infoHandle == os.Stdout {
			infoHandle = io.MultiWriter(fileHandle, infoHandle)
		}

		if warnHandle == os.Stdout {
			warnHandle = io.MultiWriter(fileHandle, warnHandle)
		}

		if errorHandle == os.Stderr {
			errorHandle = io.MultiWriter(fileHandle, errorHandle)
		}
	}

	timestamp := dateTimeUTC(log.Ldate|log.Ltime|log.Lshortfile, l.DataTimeUTC)

	logger.Debug = log.New(debugHandle, colorize("DEBUG: ", colorBlack, l.DisableColor), timestamp)
	logger.Info = log.New(infoHandle, colorize("INFO: ", colorCyan, l.DisableColor), timestamp)
	logger.Warning = log.New(warnHandle, colorize("WARNING: ", colorYellow, l.DisableColor), timestamp)
	logger.Error = log.New(errorHandle, colorize("ERROR: ", colorRed, l.DisableColor), timestamp)

	atomic.StoreInt32(&logger.LogLevel, logLevel)
}

// LogDirectoryCleanup performs all the directory cleanup and maintenance.
func (l *Logger) LogDirectoryCleanup(baseFilePath string, daysToKeep int) {

	l.Startedf("main", "LogDirectoryCleanup", "BaseFilePath[%s] DaysToKeep[%d]", baseFilePath, daysToKeep)

	// Get a list of existing directories.
	fileInfos, err := ioutil.ReadDir(baseFilePath)
	if err != nil {
		l.CompletedError("LogDirectoryCleanup", err)
		return
	}

	// Create the date to compare for directories to remove.
	currentDate := time.Now().UTC()
	compareDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day()-daysToKeep, 0, 0, 0, 0, time.UTC)

	Debug("LogDirectoryCleanup", "CompareDate[%v]", compareDate)

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() == false {
			continue
		}

		// The file name look like: YYYY-MM-DD
		parts := strings.Split(fileInfo.Name(), "-")

		year, err := strconv.Atoi(parts[0])
		if err != nil {
			Errorf("LogDirectoryCleanup", err, "Attempting To Convert Directory [%s]", fileInfo.Name())
			continue
		}

		month, err := strconv.Atoi(parts[1])
		if err != nil {
			Errorf("LogDirectoryCleanup", err, "Attempting To Convert Directory [%s]", fileInfo.Name())
			continue
		}

		day, err := strconv.Atoi(parts[2])
		if err != nil {
			Errorf("LogDirectoryCleanup", err, "Attempting To Convert Directory [%s]", fileInfo.Name())
			continue
		}

		// The directory to check.
		fullFileName := fmt.Sprintf("%s/%s", baseFilePath, fileInfo.Name())

		// Create a time type from the directory name.
		directoryDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

		// Compare the dates and convert to days.
		daysOld := int(compareDate.Sub(directoryDate).Hours() / 24)

		Debug("LogDirectoryCleanup", "Checking Directory[%s] DaysOld[%d]", fullFileName, daysOld)

		if daysOld >= 0 {
			Debug("LogDirectoryCleanup", "Removing Directory[%s]", fullFileName)

			err = os.RemoveAll(fullFileName)
			if err != nil {
				Debug("LogDirectoryCleanup", "Attempting To Remove Directory [%s]", fullFileName)
				continue
			}

			Debug("LogDirectoryCleanup", "Directory Removed [%s]", fullFileName)
		}
	}

	// We don't need the catch handler to log any errors.
	err = nil

	l.Completed("LogDirectoryCleanup")
	return
}

//** STARTED AND COMPLETED

// Started uses the Serialize destination and adds a Started tag to the log line
func (l *Logger) Started(functionName string) {
	logger.Debug.Output(2, fmt.Sprintf("%s Started\n", formatFuncName(functionName)))
}

// Startedf uses the Serialize destination and writes a Started tag to the log line
func (l *Logger) Startedf(functionName string, format string, a ...interface{}) {
	logger.Debug.Output(2, fmt.Sprintf("%s Started %s\n", formatFuncName(functionName), fmt.Sprintf(format, a...)))
}

// Completed uses the Serialize destination and writes a Completed tag to the log line
func (l *Logger) Completed(functionName string) {
	logger.Debug.Output(2, fmt.Sprintf("%s  Completed\n", formatFuncName(functionName)))
}

// Completedf uses the Serialize destination and writes a Completed tag to the log line
func (l *Logger) Completedf(functionName string, format string, a ...interface{}) {
	logger.Debug.Output(2, fmt.Sprintf("%s Completed %s\n", formatFuncName(functionName), fmt.Sprintf(format, a...)))
}

// CompletedError uses the Error destination and writes a Completed tag to the log line
func (l *Logger) CompletedError(functionName string, err error) {
	logger.Error.Output(2, fmt.Sprintf("%s Completed with ERROR : %s\n", formatFuncName(functionName), err))
}

// CompletedErrorf uses the Error destination and writes a Completed tag to the log line
func (l *Logger) CompletedErrorf(functionName string, err error, format string, a ...interface{}) {
	logger.Error.Output(2, fmt.Sprintf("%s Completed with ERROR : %s : %s\n", formatFuncName(functionName), fmt.Sprintf(format, a...), err))
}

//** DEBUG

// Debug writes to the Debug destination
func Debug(functionName string, format string, a ...interface{}) {
	logger.Debug.Output(2, fmt.Sprintf("%s %s\n", formatFuncName(functionName), fmt.Sprintf(format, a...)))
}

//** INFO

// Info writes to the Info destination
func Info(functionName string, format string, a ...interface{}) {
	logger.Info.Output(2, fmt.Sprintf("%s %s\n", formatFuncName(functionName), fmt.Sprintf(format, a...)))
}

//** WARNING

// Warning writes to the Warning destination
func Warning(functionName string, format string, a ...interface{}) {
	logger.Warning.Output(2, fmt.Sprintf("%s %s\n", formatFuncName(functionName), fmt.Sprintf(format, a...)))
}

//** ERROR

// Error writes to the Error destination and accepts an err
func Error(functionName string, err string) {
	logger.Error.Output(2, fmt.Sprintf("%s %s\n", formatFuncName(functionName), err))
}

// Errorf writes to the Error destination and accepts an err
func Errorf(functionName string, err error, format string, a ...interface{}) {
	logger.Error.Output(2, fmt.Sprintf("%s %s %s\n", formatFuncName(functionName), fmt.Sprintf(format, a...), err))
}

// colorize the log out put based on the need
func colorize(s interface{}, c int, disableColor bool) string {
	if disableColor {
		return fmt.Sprintf("%s", s)
	}
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

// options to use UTC timestamps
func dateTimeUTC(i int, useUTC bool) int {
	if useUTC {
		return i | log.LUTC
	}
	return i
}

// used to format function name expected func name to be funcName()
// if the trailing braces are absent adding it
func formatFuncName(s string) string {
	matched, err := regexp.MatchString(`*()`, s)
	if err != nil {
		// Error("formatFuncName", fmt.Sprintf("Error in regexp matching: %v", err))
		log.Printf("Error: %v\n", err)
	}
	if matched {
		return s
	}
	return fmt.Sprintf("%s()", s)
}
