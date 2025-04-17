package utils

import (
	"fmt"
	"log"
	"os"
)

type ColoredLogger struct {
	*log.Logger
	color string
}

func NewColoredLogger(prefix string, color string) *ColoredLogger {
	return &ColoredLogger{
		Logger: log.New(os.Stdout, fmt.Sprintf("%s[%s]%s ", color, prefix, "\033[0m"), log.Ldate|log.Ltime),
		color:  color,
	}
}

func (l *ColoredLogger) Printf(format string, v ...interface{}) {
	l.Logger.Printf(format, v...)
}

func (l *ColoredLogger) Println(v ...interface{}) {
	l.Logger.Println(v...)
}
