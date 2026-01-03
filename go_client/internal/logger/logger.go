package logger

import "fmt"

func Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func Info(args ...interface{}) {
	fmt.Print("[INFO] ")
	fmt.Println(args...)
}

func Warnf(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

func Warn(args ...interface{}) {
	fmt.Print("[WARN] ")
	fmt.Println(args...)
}

func Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func Debugf(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

func Init(logDir string) error {
	fmt.Printf("[Logger] Initialized in %s\n", logDir)
	return nil
}

func Tail(lines int) ([]string, error) {
	// Stub for now
	return []string{"Log Line 1", "Log Line 2"}, nil
}
