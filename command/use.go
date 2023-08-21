package command

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"

	"github.com/shyiko/jabba/cfg"
	"golang.org/x/sys/windows"
)

func Use(selector string) ([]string, error) {
	aliasValue := GetAlias(selector)
	if aliasValue != "" {
		selector = aliasValue
	}
	ver, err := LsBestMatch(selector)
	if err != nil {
		return nil, err
	}
	return usePath(filepath.Join(cfg.Dir(), "jdk", ver))
}

func usePath(path string) ([]string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	pth, _ := os.LookupEnv("PATH")
	rgxp := regexp.MustCompile(regexp.QuoteMeta(filepath.Join(cfg.Dir(), "jdk")) + "[^:]+[:]")
	// strip references to ~/.jabba/jdk/*, otherwise leave unchanged
	pth = rgxp.ReplaceAllString(pth, "")
	if runtime.GOOS == "darwin" {
		path = filepath.Join(path, "Contents", "Home")
	}

	if runtime.GOOS == "windows" {
		shouldReturn, returnValue, returnValue1 := changeJavaVersionForWindows(path)
		if shouldReturn {
			return returnValue, returnValue1
		}
	}
	systemJavaHome, overrideWasSet := os.LookupEnv("JAVA_HOME_BEFORE_JABBA")
	if !overrideWasSet {
		systemJavaHome, _ = os.LookupEnv("JAVA_HOME")
	}
	return []string{
		"export PATH=\"" + filepath.Join(path, "bin") + string(os.PathListSeparator) + pth + "\"",
		"export JAVA_HOME=\"" + path + "\"",
		"export JAVA_HOME_BEFORE_JABBA=\"" + systemJavaHome + "\"",
	}, nil
}

func changeJavaVersionForWindows(path string) (bool, []string, error) {
	if !amAdmin() {
		runMeElevated()
	}
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("It seems you dont have a home directory")
		return true, nil, err
	} else {

		var destFile *os.File
		windowsProfile := filepath.Join(homedir, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		_, err := os.Stat(windowsProfile)
		if err == nil {
			destFile, err = os.OpenFile(windowsProfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				fmt.Println("An error occured while accesing the windows profile", err)
				return true, nil, err
			}
		} else if os.IsNotExist(err) {
			destFile, err = os.Create(windowsProfile)
			if err != nil {
				fmt.Println("An error occured while creating the windows profile", err)
				return true, nil, err
			}

		} else {
			fmt.Printf("It seems you dont have a home directory")
			return true, nil, err
		}

		wPContent, err := os.ReadFile(windowsProfile)
		if err != nil {
			fmt.Println("Error:", err)
			return true, nil, err
		}

		ver := filepath.Base(path)
		wPContentStr := string(wPContent)

		regexPattern := `jabba use .+`
		replacementValue := fmt.Sprintf("jabba use %s", ver)
		re := regexp.MustCompile(regexPattern)
		isMatch := re.MatchString(wPContentStr)
		var newWPContentStr string
		if isMatch {
			newWPContentStr = re.ReplaceAllString(wPContentStr, replacementValue)
		} else {
			newWPContentStr = fmt.Sprintf("%s\n%s", wPContentStr, replacementValue)
		}
		fmt.Print(newWPContentStr)

		_, err = destFile.WriteString(newWPContentStr)
		if err != nil {
			fmt.Println("An error occured while updating the profile")
			return true, nil, err
		}

		defer destFile.Close()

	}
	return false, nil, nil
}

func runMeElevated() {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 //SW_NORMAL

	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		fmt.Println(err)
	}
}

func amAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		fmt.Println("admin no")
		return false
	}
	fmt.Println("admin yes")
	return true
}
