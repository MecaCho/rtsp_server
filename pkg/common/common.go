package common

import (
	"fmt"
	"strings"
	"os"
	"time"
)



//GetClientID get mqtt broker clientID
func GetClientID() string {
	timeNow := time.Now().String()
	tmpClient := []string{"camera_checker", getHostName(), timeNow}
	return strings.Join(tmpClient, "#")
}

func getHostName() string {
	ret, err := os.Hostname()
	if err != nil {
		return ""
	}
	return ret
}


//FormatLog gen format log output
func FormatLog(logStirng string) string {
	if len(logStirng) > 79 {
		lens := len(logStirng) / 80
		logLines := []string{}
		logLines = append(logLines, "||"+logStirng[:80]+"||")
		for i := 1; i < lens; i++ {
			logLines = append(logLines, "                                                     ||"+logStirng[i*80:i*80+80]+"||")
		}
		logLines = append(logLines, fmt.Sprintf("                                                     ||%-80s||", logStirng[lens*80:]))
		fmtLog := strings.Join(logLines, "\n")
		return fmtLog
	}
	fmtLog := fmt.Sprintf("||%-80s||", logStirng)
	return fmtLog
}
