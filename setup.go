package main
 
import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)
 
// Sections whose entries are free-form "name = value" lists. In these the
// user can remove entries ("-") and add new ones.
var listSections = map[string]bool{"frequencies": true, "messages": true}
 
// Keys that must be whole numbers.
var intKeys = map[string]bool{
	"port": true, "min_dwell": true, "max_dwell": true, "quiet_time": true,
	"offset": true, "min_interval": true,
}
 
type setupUI struct {
	in *bufio.Reader
}
 
func (u *setupUI) readLine() (string, error) {
	s, err := u.in.ReadString('\n')
	if err != nil && !(err == io.EOF && s != "") {
		return "", err
	}
	return strings.TrimSpace(s), nil
}
 
func (u *setupUI) yesNo(q string, def bool) (bool, error) {
	d := "y/N"
	if def {
		d = "Y/n"
	}
	for {
		fmt.Printf("%s [%s]: ", q, d)
		s, err := u.readLine()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(s) {
		case "":
			return def, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
	}
}
 
func validateSetting(sec, key, val string) error {
	switch {
	case sec == "frequencies":
		if _, err := parseFreq(val); err != nil {
			return fmt.Errorf("not a frequency (use Hz like 7078000 or MHz like 7.078)")
		}
	case intKeys[key]:
		if _, err := strconv.Atoi(val); err != nil {
			return fmt.Errorf("must be a whole number")
		}
	}
	return nil
}
 
// normalize turns y/n style answers into true/false when the default is boolean.
func normalize(def, ans string) string {
	if d := strings.ToLower(def); d == "true" || d == "false" {
		switch strings.ToLower(ans) {
		case "y", "yes", "true", "1", "on":
			return "true"
		case "n", "no", "false", "0", "off":
			return "false"
		}
	}
	return ans
}
 
// ask shows one setting and returns the accepted or overridden value.
// remove is true if the user typed "-" (list sections only).
func (u *setupUI) ask(sec, key, def string, list bool) (val string, remove bool, err error) {
	hint := "Enter = keep"
	if list {
		hint += ", - = remove"
	}
	for {
		fmt.Printf("  %s [%s]  (%s): ", key, def, hint)
		s, e := u.readLine()
		if e != nil {
			return "", false, e
		}
		if s == "" {
			return def, false, nil
		}
		if list && s == "-" {
			return "", true, nil
		}
		s = normalize(def, s)
		if ve := validateSetting(sec, key, s); ve != nil {
			fmt.Printf("  ! %v\n", ve)
			continue
		}
		return s, false, nil
	}
}
 
// RunSetup walks through examplePath, prompts for every setting, and writes
// the result (keeping the example's comments) to outPath.
func RunSetup(examplePath, outPath string) error {
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", examplePath, err)
	}
	ui := &setupUI{in: bufio.NewReader(os.Stdin)}
 
	if _, err := os.Stat(outPath); err == nil {
		ok, err := ui.yesNo(outPath+" already exists. Overwrite?", false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Setup cancelled; nothing changed.")
			return nil
		}
	}
 
	fmt.Println("js8scan setup")
	fmt.Println("For each setting, press Enter to accept the default or type a new value.")
 
	var out []string
	var help []string
	sec := ""
	lastKV := -1 // index in out of the last key=value line
 
	flushList := func() error {
		if !listSections[sec] {
			return nil
		}
		fmt.Printf("\nAdd more entries to [%s]? Enter a name (blank to finish).\n", sec)
		var extra []string
		for {
			fmt.Print("  new name: ")
			name, err := ui.readLine()
			if err != nil {
				return err
			}
			if name == "" {
				break
			}
			name = strings.ToLower(name)
			if strings.ContainsAny(name, "=;#[]") {
				fmt.Println("  ! name can't contain = ; # [ ]")
				continue
			}
			v, _, err := ui.ask(sec, name, "", false)
			if err != nil {
				return err
			}
			if v == "" {
				fmt.Println("  (skipped: empty value)")
				continue
			}
			extra = append(extra, name+" = "+v)
		}
		if len(extra) > 0 {
			at := lastKV + 1
			if lastKV < 0 {
				at = len(out)
			}
			out = append(out[:at], append(extra, out[at:]...)...)
		}
		return nil
	}
 
	for _, raw := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		t := strings.TrimSpace(raw)
		switch {
		case t == "":
			out = append(out, raw)
			help = nil
		case t[0] == ';' || t[0] == '#':
			out = append(out, raw)
			help = append(help, strings.TrimSpace(strings.TrimLeft(t, ";# ")))
		case t[0] == '[' && t[len(t)-1] == ']':
			if err := flushList(); err != nil {
				return err
			}
			sec = strings.ToLower(strings.TrimSpace(t[1 : len(t)-1]))
			out = append(out, raw)
			lastKV = -1
			fmt.Printf("\n== [%s] ==\n", sec)
			for _, h := range help {
				fmt.Printf("  # %s\n", h)
			}
			help = nil
		default:
			i := strings.IndexByte(t, '=')
			if i < 0 {
				out = append(out, raw)
				continue
			}
			key := strings.TrimSpace(t[:i])
			def := strings.TrimSpace(t[i+1:])
			for _, h := range help {
				fmt.Printf("  # %s\n", h)
			}
			help = nil
			v, remove, err := ui.ask(sec, strings.ToLower(key), def, listSections[sec])
			if err != nil {
				return err
			}
			if remove {
				continue
			}
			out = append(out, key+" = "+v)
			lastKV = len(out) - 1
		}
	}
	if err := flushList(); err != nil {
		return err
	}
 
	if err := os.WriteFile(outPath, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("\nWrote %s\n", outPath)
 
	if _, err := LoadConfig(outPath); err != nil {
		fmt.Printf("Warning: the new file didn't pass validation: %v\n", err)
		fmt.Println("Edit it by hand or run setup again.")
	} else {
		fmt.Println("Config validated OK.")
	}
	return nil
}
