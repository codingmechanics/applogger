# applogger

applogger wraps [Go's log library](https://golang.org/src/log/log.go). A logger implementation that is very simple to use. 

## Usage 
### Get Started
Download and install it:

`go get github.com/codingmechanics/applogger`

Import it in your code:

`import "github.com/codingmechanics/applogger"`

### Basic Example
```
package main

import (
    "fmt"
    "errors"

    "github.com/codingmechanics/applogger"
)

func main() {
    // initiate the loogger
    log := applogger.Logger{}
    // start the logger. set the loglevel application wise
    // set to debug level
    log.Start(applogger.LevelDebug)
    // applicaiton code goes here
    Example()
    // stop the logger
    log.Stop()
}

//  dummy function
func Example() {
    log.Started("Example")

    log.Debug("Example", "Debug Log")
    log.Info("Example", "Info Log")
    log.Warning("Example", "Warn Log")
    log.Error("Example()", "Error Log", error.New("Dummy Error"))

    log.Completed("Example")
}
```

Output 

```
DEBUG: 2019/10/31 20:26:10 main.go:73: Example() Started
INFO: 2019/10/31 20:26:10 main.go:76: Example() Info Log
WARNING: 2019/10/31 20:26:10 main.go:77: Example() Warn Log
ERROR: 2019/10/31 20:26:10 main.go:78: Example() Error Log: Dummy Error
DEBUG: 2019/10/31 20:26:10 main.go:80: Example()  Completed
```
