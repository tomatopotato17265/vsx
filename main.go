package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var SupportedIDEs map[string][]string

func init() {
	home, _ := os.UserHomeDir()
	localAppData := filepath.Join(home, "AppData", "Local")
	
	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		programFiles = `C:\Program Files`
	}
	
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	if programFilesX86 == "" {
		programFilesX86 = `C:\Program Files (x86)`
	}

	SupportedIDEs = map[string][]string{
		"Cursor_darwin":             {"/Applications/Cursor.app/Contents/Resources/app/bin/cursor"},
		"Antigravity_darwin":        {"/Applications/Antigravity.app/Contents/Resources/app/bin/antigravity"},
		"Visual Studio Code_darwin": {"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"},
		"VSCodium_darwin":           {"/Applications/VSCodium.app/Contents/Resources/app/bin/codium"},

		"Cursor_linux":             {"/usr/local/bin/cursor", "/usr/bin/cursor"},
		"Antigravity_linux":        {"/usr/local/bin/antigravity", "/usr/bin/antigravity"},
		"Visual Studio Code_linux": {"/usr/bin/code", "/usr/share/code/bin/code"},
		"VSCodium_linux":           {"/usr/bin/codium", "/usr/share/codium/bin/codium"},

		"Cursor_windows": {
			filepath.Join(localAppData, "Programs", "cursor", "resources", "app", "bin", "cursor.cmd"),
			filepath.Join(programFiles, "cursor", "resources", "app", "bin", "cursor.cmd"),
			filepath.Join(localAppData, "Programs", "cursor", "bin", "cursor.cmd"),
			filepath.Join(programFiles, "cursor", "bin", "cursor.cmd"),
		},
		"Antigravity_windows": {
			filepath.Join(localAppData, "Programs", "antigravity", "resources", "app", "bin", "antigravity.cmd"),
			filepath.Join(programFiles, "antigravity", "resources", "app", "bin", "antigravity.cmd"),
			filepath.Join(localAppData, "Programs", "antigravity", "bin", "antigravity.cmd"),
			filepath.Join(programFiles, "antigravity", "bin", "antigravity.cmd"),
		},
		"Visual Studio Code_windows": {
			filepath.Join(localAppData, "Programs", "Microsoft VS Code", "bin", "code.cmd"),
			filepath.Join(programFiles, "Microsoft VS Code", "bin", "code.cmd"),
			filepath.Join(programFilesX86, "Microsoft VS Code", "bin", "code.cmd"),
		},
		"VSCodium_windows": {
			filepath.Join(localAppData, "Programs", "VSCodium", "bin", "codium.cmd"),
			filepath.Join(programFiles, "VSCodium", "bin", "codium.cmd"),
			filepath.Join(programFilesX86, "VSCodium", "bin", "codium.cmd"),
			filepath.Join(localAppData, "Programs", "VSCodium", "resources", "app", "bin", "codium.cmd"),
			filepath.Join(programFiles, "VSCodium", "resources", "app", "bin", "codium.cmd"),
		},
	}
}

func logDebug(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".vsx.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, msg))
		f.Close()
	}
}

type Config struct {
	DefaultIDEs []string `json:"defaultIDEs"`
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".vsx.json")
}

func loadConfig() Config {
	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		return Config{}
	}
	var config Config
	json.Unmarshal(data, &config)
	return config
}

func saveConfig(ideNames []string) {
	config := Config{DefaultIDEs: ideNames}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(getConfigPath(), data, 0644)
}

func detectInstalledIDEs() map[string]string {
	installed := make(map[string]string)
	for name, paths := range SupportedIDEs {
		if !strings.HasSuffix(name, "_"+runtime.GOOS) {
			continue
		}

		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				cleanName := strings.TrimSuffix(name, "_"+runtime.GOOS)
				installed[cleanName] = path
				break
			}
		}
	}
	return installed
}

func promptUser(installed map[string]string) []string {
	var names []string
	for name := range installed {
		names = append(names, name)
	}

	if runtime.GOOS == "darwin" {
		listStr := `{"` + strings.Join(names, `", "`) + `"}`
		
		script := fmt.Sprintf(`
		activate
		set theChoice to choose from list %s with prompt "VSX found multiple IDEs. Select where to install this extension:" with multiple selections allowed
		if theChoice is false then
			return "false"
		else
			set AppleScript's text item delimiters to ", "
			return theChoice as string
		end if
		`, listStr)

		out, err := exec.Command("osascript", "-e", script).Output()
		choice := strings.TrimSpace(string(out))

		if err != nil || choice == "false" || choice == "" {
			logDebug("Prompt cancelled or failed. Error: %v", err)
			return nil
		}
		return strings.Split(choice, ", ")

	} else if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("zenity"); err == nil {
			args := []string{"--list", "--checklist", "--title=VSX IDE Selection", "--text=Select where to install this extension:", "--column=Select", "--column=IDE"}
			for _, name := range names {
				args = append(args, "FALSE", name)
			}
			out, err := exec.Command("zenity", args...).Output()
			if err != nil {
				logDebug("Zenity prompt failed or cancelled: %v", err)
				return nil
			}
			choice := strings.TrimSpace(string(out))
			if choice == "" {
				return nil
			}
			return strings.Split(choice, "|")
			
		} else if _, err := exec.LookPath("kdialog"); err == nil {
			args := []string{"--title", "VSX IDE Selection", "--checklist", "Select where to install this extension:"}
			for _, name := range names {
				args = append(args, name, name, "off")
			}
			out, err := exec.Command("kdialog", args...).Output()
			if err != nil {
				logDebug("Kdialog prompt failed or cancelled: %v", err)
				return nil
			}
			choice := strings.TrimSpace(string(out))
			if choice == "" {
				return nil
			}
			
			var selected []string
			parts := strings.Split(choice, `" "`)
			for _, p := range parts {
				clean := strings.Trim(p, `"`)
				if clean != "" {
					selected = append(selected, clean)
				}
			}
			return selected
			
		} else {
			logDebug("No Linux GUI dialog tool found. Defaulting to all.")
			return names
		}

	} else if runtime.GOOS == "windows" {
		psScript := `
		Add-Type -AssemblyName System.Windows.Forms
		Add-Type -AssemblyName System.Drawing
		
		[System.Windows.Forms.Application]::EnableVisualStyles()
		
		$form = New-Object System.Windows.Forms.Form
		$form.Text = "VSX IDE Selection"
		$form.Size = New-Object System.Drawing.Size(300,230)
		$form.StartPosition = "CenterScreen"
		$form.TopMost = $true
		
		$form.FormBorderStyle = "None"
		$form.BackColor = [System.Drawing.SystemColors]::Control
		
		$label = New-Object System.Windows.Forms.Label
		$label.Text = "Select where to install this extension:"
		$label.Location = New-Object System.Drawing.Point(15,15)
		$label.AutoSize = $true
		$form.Controls.Add($label)
		
		$listBox = New-Object System.Windows.Forms.ListBox
		$listBox.Location = New-Object System.Drawing.Point(15,40)
		$listBox.Size = New-Object System.Drawing.Size(270,120)
		$listBox.SelectionMode = "MultiExtended"
		`
		for _, name := range names {
			psScript += fmt.Sprintf(`$listBox.Items.Add("%s") | Out-Null`+"\n", name)
		}
		psScript += `
		$form.Controls.Add($listBox)
		
		$okButton = New-Object System.Windows.Forms.Button
		$okButton.Location = New-Object System.Drawing.Point(115,180)
		$okButton.Text = "OK"
		$okButton.Size = New-Object System.Drawing.Size(75,25)
		$okButton.DialogResult = [System.Windows.Forms.DialogResult]::OK
		$form.Controls.Add($okButton)
		
		$cancelButton = New-Object System.Windows.Forms.Button
		$cancelButton.Location = New-Object System.Drawing.Point(200,180)
		$cancelButton.Text = "Cancel"
		$cancelButton.Size = New-Object System.Drawing.Size(75,25)
		$cancelButton.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
		$form.Controls.Add($cancelButton)
		
		$form.AcceptButton = $okButton
		$form.CancelButton = $cancelButton
		
		$result = $form.ShowDialog()
		
		if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
			$listBox.SelectedItems | ForEach-Object { Write-Output $_ }
		}
		`
		
		out, err := runHiddenPowershell(psScript)
		if err != nil {
			logDebug("Windows prompt failed: %v", err)
			return nil
		}

		outputStr := strings.TrimSpace(string(out))
		if outputStr == "" {
			return nil
		}
		
		var selected []string
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			cleanLine := strings.TrimSpace(line)
			if cleanLine != "" {
				selected = append(selected, cleanLine)
			}
		}
		return selected
	}

	return names
}

func main() {
	if len(os.Args) < 2 {
		return
	}
	arg := os.Args[1]

	if arg == "default" {
		installed := detectInstalledIDEs()
		if len(installed) == 0 {
			fmt.Println("No compatible IDEs found on this system.")
			return
		}

		fmt.Println("Installed IDEs:")
		var names []string
		for name := range installed {
			names = append(names, name)
		}

		for i, name := range names {
			fmt.Printf("[%d] %s\n", i+1, name)
		}

		fmt.Print("\nEnter the numbers of the IDEs you want to set as default (e.g., 1, 2): ")
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Println("No selection made. Defaults not updated.")
			return
		}

		input = strings.ReplaceAll(input, " ", ",")
		parts := strings.Split(input, ",")
		
		var selectedDefaults []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			idx, err := strconv.Atoi(p)
			if err == nil && idx >= 1 && idx <= len(names) {
				ideName := names[idx-1]
				duplicate := false
				for _, existing := range selectedDefaults {
					if existing == ideName {
						duplicate = true
						break
					}
				}
				if !duplicate {
					selectedDefaults = append(selectedDefaults, ideName)
				}
			}
		}

		if len(selectedDefaults) > 0 {
			saveConfig(selectedDefaults)
			fmt.Printf("Successfully set default IDE(s) to: %s\n", strings.Join(selectedDefaults, ", "))
		} else {
			fmt.Println("No valid selections made. Defaults not updated.")
		}
		return
	}

	if arg == "setup" {
		if runtime.GOOS == "linux" {
			setupLinux()
		} else if runtime.GOOS == "windows" {
			setupWindows()
		} else if runtime.GOOS == "darwin" {
			setupMac()
		}
		return
	}

	logDebug("--- VSX Triggered ---")
	logDebug("Raw URL received: %s", arg)

	parsedArg := strings.TrimSpace(arg)
	parsedArg = strings.TrimPrefix(parsedArg, "vscode://")
	parsedArg = strings.TrimPrefix(parsedArg, "vscode:")
	parsedArg = strings.TrimLeft(parsedArg, "/") 

	if strings.HasPrefix(parsedArg, "extension/") {
		extId := strings.TrimPrefix(parsedArg, "extension/")
		logDebug("Parsed Extension ID: %s", extId)

		installed := detectInstalledIDEs()
		if len(installed) == 0 {
			logDebug("Error: No compatible IDEs found.")
			return
		}

		var targetNames []string
		config := loadConfig()

		if len(config.DefaultIDEs) > 0 {
			for _, def := range config.DefaultIDEs {
				if _, exists := installed[def]; exists {
					targetNames = append(targetNames, def)
				}
			}
		}

		if len(targetNames) == 0 {
			logDebug("No default configured. Triggering OS prompt.")
			if len(installed) == 1 {
				for name := range installed {
					targetNames = append(targetNames, name)
				}
			} else {
				targetNames = promptUser(installed)
				if len(targetNames) == 0 {
					logDebug("User cancelled the prompt.")
					return
				}
			}
		}

		logDebug("Target IDEs selected: %v", targetNames)

		for _, targetName := range targetNames {
			cmdPath := installed[targetName]
			logDebug("Executing install via: %s", cmdPath)
			
			err := installExtension(cmdPath, extId)

			if err != nil {
				logDebug("INSTALLATION FAILED for %s: %v", targetName, err)
			} else {
				logDebug("Successfully installed to %s!", targetName)
			}
		}
	} else {
		logDebug("URL did not match extension routing prefix.")
	}
}

func setupMac() {
	exePath, _ := os.Executable()
	home, _ := os.UserHomeDir()
	
	userAppsDir := filepath.Join(home, "Applications")
	os.MkdirAll(userAppsDir, 0755)
	
	appDir := filepath.Join(userAppsDir, "VSX.app")
	plistPath := filepath.Join(appDir, "Contents", "Info.plist")

	script := fmt.Sprintf(`on open location this_url
		set binPath to POSIX path of (path to me) & "Contents/MacOS/vsx"
		do shell script quoted form of binPath & " " & quoted form of this_url & " >> %s/.vsx.log 2>&1"
	end open location`, home)

	tmpScript := filepath.Join(os.TempDir(), "router.applescript")
	os.WriteFile(tmpScript, []byte(script), 0644)
	defer os.Remove(tmpScript)

	os.RemoveAll(appDir) 
	
	err := exec.Command("osacompile", "-x", "-o", appDir, tmpScript).Run()
	if err != nil {
		fmt.Printf("Failed to generate macOS app wrapper. Error: %v\n", err)
		return
	}

	macOSDir := filepath.Join(appDir, "Contents", "MacOS")
	os.MkdirAll(macOSDir, 0755)
	
	exeBytes, err := os.ReadFile(exePath)
	if err == nil {
		err = os.WriteFile(filepath.Join(macOSDir, "vsx"), exeBytes, 0755)
		if err != nil {
			fmt.Printf("Error writing binary to bundle: %v\n", err)
		}
	} else {
		fmt.Printf("Failed to read binary for App Bundle injection: %v\n", err)
		return
	}

	exec.Command("/usr/libexec/PlistBuddy", "-c", `Set :CFBundleIdentifier com.vsx.router`, plistPath).Run()
	exec.Command("/usr/libexec/PlistBuddy", "-c", `Add :CFBundleIdentifier string com.vsx.router`, plistPath).Run()
	exec.Command("/usr/libexec/PlistBuddy", "-c", `Add :CFBundleShortVersionString string 99.9.9`, plistPath).Run()
	exec.Command("/usr/libexec/PlistBuddy", "-c", `Set :CFBundleShortVersionString 99.9.9`, plistPath).Run()

	plistCmds := []string{
		`Add :CFBundleURLTypes array`,
		`Add :CFBundleURLTypes:0 dict`,
		`Add :CFBundleURLTypes:0:CFBundleURLName string "Visual Studio Code Extension"`,
		`Add :CFBundleURLTypes:0:CFBundleURLSchemes array`,
		`Add :CFBundleURLTypes:0:CFBundleURLSchemes:0 string "vscode"`,
		`Add :LSUIElement bool true`,
	}
	for _, cmd := range plistCmds {
		exec.Command("/usr/libexec/PlistBuddy", "-c", cmd, plistPath).Run()
	}

	exec.Command("touch", plistPath).Run()
	exec.Command("touch", appDir).Run()

	exec.Command("/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister", "-f", appDir).Run()

	os.Remove(filepath.Join(home, ".vsx.log"))

	fmt.Println("macOS Protocol Registered successfully!")
}

func setupLinux() {
	exePath, _ := os.Executable()
	home, _ := os.UserHomeDir()

	desktopFile := fmt.Sprintf(`[Desktop Entry]
Name=VSX
Exec=%s %%u
Type=Application
NoDisplay=true
MimeType=x-scheme-handler/vscode;`, exePath)

	path := filepath.Join(home, ".local", "share", "applications", "vsx.desktop")
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(desktopFile), 0644)

	exec.Command("xdg-mime", "default", "vsx.desktop", "x-scheme-handler/vscode").Run()
	fmt.Println("Linux Protocol Registered!")
}

func setupWindows() {
	exePath, _ := os.Executable()
	commands := [][]string{
		{"reg", "add", "HKCU\\Software\\Classes\\vscode", "/ve", "/d", "URL:vscode", "/f"},
		{"reg", "add", "HKCU\\Software\\Classes\\vscode", "/v", "URL Protocol", "/d", "", "/f"},
		{"reg", "add", "HKCU\\Software\\Classes\\vscode\\shell\\open\\command", "/ve", "/d", fmt.Sprintf("\"%s\" \"%%1\"", exePath), "/f"},
	}

	for _, args := range commands {
		exec.Command(args[0], args[1:]...).Run()
	}
	fmt.Println("Windows Protocol Registered!")
}