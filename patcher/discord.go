package patcher

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/phips4/discord-update-patcher/zip"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Discord struct {
	dir        string
	Version    string
	modulesDir string
}

type DiscordModules map[string]int

func (discord *Discord) SelectDefaultPath() {
	discord.dir = path.Join(userHomeDir(), "AppData", "Roaming", "discord")

	if _, err := os.Stat(discord.dir); os.IsNotExist(err) {
		fmt.Println("could not find default discord installation directory.")
		return
	}

	fmt.Println("Selected default installation directory:", discord.dir)
}

func (discord *Discord) SelectLatestVersion() {
	files, err := ioutil.ReadDir(discord.dir)
	must(err)
	var latestVersion string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "0.") { //probably changes in the future
			latestVersion = file.Name()
		}
	}
	discord.Version = latestVersion
	fmt.Println("Selected version:", discord.Version)
}

func (discord *Discord) SelectModulesPath(version string) {
	discord.modulesDir = path.Join(discord.dir, version, "modules")
	fmt.Println("Selected modules directory:", discord.modulesDir)
}

func (discord *Discord) CreateBackup() {
	fmt.Println("Creating backup...")
	fileFormat := fmt.Sprintf("DUP_%d_%s.zip", time.Now().Unix(), discord.Version)

	src := filepath.Join(discord.modulesDir, "..")
	dest := filepath.Join(discord.modulesDir, "..", "..", "dup-backup", fileFormat)

	// creates a zipped copy of src
	must(zip.Zip(src, dest))
}

func (discord *Discord) DeleteModules() {
	files, err := ioutil.ReadDir(discord.modulesDir)
	must(err)

	fmt.Println("Deleting modules...")
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "discord_") || !file.IsDir() {
			continue
		}
		if file.Name() == "discord_desktop_core" { // exclude module in order to be able to start the updater
			continue
		}

		 if err := os.RemoveAll(path.Join(discord.modulesDir, file.Name())); err != nil {
		 	fmt.Println("Is discord still running?")
		 	log.Fatalln(err)
		 }
	}
	discord.mustRemoveFromModules("pending")
	discord.mustRemoveFromModules("installed.json")
}

func (discord *Discord) mustRemoveFromModules(target string) {
	must(os.RemoveAll(path.Join(discord.modulesDir, target)))
}

func (discord *Discord) DownloadFiles() {
	exePath := validateExePath( // default discord installation dir with version, if not ok ask user to enter location
		path.Join(discord.dir, "..", "..", "local", "Discord", "app-"+discord.Version, "Discord.exe"))

	fmt.Println("Selected Discord.exe:", exePath)
	fmt.Println("Starting discord to download Files")

	cmd := exec.Command(exePath, "")
	cmdReader, err := cmd.StdoutPipe()
	must(err)

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Finished module downloads") {
				fmt.Println("Download finished now kill discord")
				must(cmd.Process.Kill())
			}
		}
	}()

	must(cmd.Start())
	err = cmd.Wait()

	// exit status 1 is okay, because we kill the process. Other errors should be handled
	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		must(err)
	}
}

func validateExePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) || !strings.HasSuffix(path, "Discord.exe") {
		path = readExeInput()
		validateExePath(path)
	}
	return path
}

func readExeInput() string {
	fmt.Println("Discord.exe could not be found. Please enter location manually:")
	exePath, err := bufio.NewReader(os.Stdin).ReadString('\n')
	must(err)
	return strings.TrimSuffix(exePath, "\n")
}

func userHomeDir() string {
	if runtime.GOOS != "windows" {
		return os.Getenv("HOME")
	}
	home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	return home
}

func (discord *Discord) InstallModules() DiscordModules {
	discord.mustRemoveFromModules("discord_desktop_core")

	modules, err := unzipModules(filepath.Join(discord.modulesDir, "pending"))
	must(err)

	discord.mustRemoveFromModules("pending")
	return modules
}

func unzipModules(pendingPath string) (DiscordModules, error) {
	modules := make(DiscordModules)
	err := filepath.Walk(pendingPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		fmt.Println("installing module", info.Name())

		moduleName, version, err := extractModuleName(info.Name())
		modules[moduleName] = version

		modulesPath := filepath.Join(pendingPath, "..")
		moduleDir := filepath.Join(modulesPath, moduleName)

		if _, err := os.Stat(moduleDir); !os.IsNotExist(err) {
			if err = os.RemoveAll(moduleDir); err != nil {
				return nil
			}
		}

		if err = zip.Unzip(filepath.Join(pendingPath, info.Name()), moduleDir); err != nil {
			return err
		}
		return nil
	})
	return modules, err
}

func extractModuleName(moduleName string) (string, int, error) {
	moduleName = strings.TrimSuffix(moduleName, ".zip")
	split := strings.Split(moduleName, "-")

	moduleName = split[0]
	version, err := strconv.Atoi(split[1])
	if err != nil {
		return "", -1, err
	}

	return moduleName, version, nil
}

func (discord *Discord) UpdateJson(modules DiscordModules) {
	discord.manipulateJson(modules)
}

func (discord *Discord) manipulateJson(modules DiscordModules) {
	installedFile, err := os.OpenFile(path.Join(discord.modulesDir, "installed.json"), os.O_RDWR, os.ModePerm)
	must(err)
	defer installedFile.Close()

	jsonMap := map[string]json.RawMessage{}
	must(json.NewDecoder(installedFile).Decode(&jsonMap))

	type moduleDetails struct {
		InstalledVersion int    `json:"installedVersion"`
		UpdateVersion    int    `json:"updateVersion"`
		UpdateZipFile    string `json:"updateZipfile,omitempty"`
	}

	finalJson := make(map[string]moduleDetails)

	for moduleName := range jsonMap {
		details := &moduleDetails{}
		must(json.Unmarshal(jsonMap[moduleName], details))

		details.InstalledVersion = modules[moduleName]
		details.UpdateZipFile = ""
		finalJson[moduleName] = *details
		fmt.Println("update module version", moduleName, modules[moduleName])
	}

	finalJsonBytes, err := json.MarshalIndent(finalJson, "", "  ")
	must(err)

	must(installedFile.Truncate(0))

	_, err = installedFile.WriteAt(finalJsonBytes, 0)
	must(err)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
