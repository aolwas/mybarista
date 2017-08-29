package main

import (
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
  "github.com/soumya92/barista/modules/battery"
  "github.com/soumya92/barista/modules/clock"
	"github.com/soumya92/barista/modules/group"
	"github.com/soumya92/barista/modules/meminfo"
	"github.com/soumya92/barista/modules/sysinfo"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/pango/icons/fontawesome"
	"github.com/soumya92/barista/pango/icons/ionicons"
	"github.com/soumya92/barista/pango/icons/material"
	"github.com/soumya92/barista/pango/icons/material_community"
	"github.com/soumya92/barista/pango/icons/typicons"
)

var spacer = pango.Span(" ", pango.XXSmall)

func truncate(in string, l int) string {
	if len([]rune(in)) <= l {
		return in
	}
	return string([]rune(in)[:l-1]) + "⋯"
}

func hms(d time.Duration) (h int, m int, s int) {
	h = int(d.Hours())
	m = int(d.Minutes()) % 60
	s = int(d.Seconds()) % 60
	return
}

func formatMediaTime(d time.Duration) string {
	h, m, s := hms(d)
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func startTaskManager(e bar.Event) {
	if e.Button == bar.ButtonLeft {
		exec.Command("gnome-system-monitor").Run()
	}
}

func home(path string) string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	return filepath.Join(usr.HomeDir, path)
}

func main() {
	material.Load(home("Github/material-design-icons"))
	materialCommunity.Load(home("Github/MaterialDesign-Webfont"))
	typicons.Load(home("Github/typicons.font"))
	ionicons.Load(home("Github/ionicons"))
	fontawesome.Load(home("Projects/Perso/Font-Awesome"))

	colors.LoadFromMap(map[string]string{
		"good":     "#6d6",
		"degraded": "#dd6",
		"bad":      "#d66",
		"dim-icon": "#777",
	})

	localtime := clock.New().OutputFunc(func(now time.Time) bar.Output {
		return outputs.Pango(
			fontawesome.Icon("calendar-o", colors.Scheme("dim-icon")),
			spacer,
            now.Format("Jan 2"),
			spacer,
            fontawesome.Icon("clock-o", colors.Scheme("dim-icon")),
		    spacer,
            now.Format(" 15:04:05"),
		)
	}).OnClick(func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			exec.Command("gsimplecal").Run()
		}
	})

	loadAvg := sysinfo.New().OutputFunc(func(s sysinfo.Info) bar.Output {
		out := outputs.Textf("%0.2f %0.2f", s.Loads[0], s.Loads[2])
		// Load averages are unusually high for a few minutes after boot.
		if s.Uptime < 10*time.Minute {
			// so don't add colours until 10 minutes after system start.
			return out
		}
		switch {
		case s.Loads[0] > 128, s.Loads[2] > 64:
			out.Urgent(true)
		case s.Loads[0] > 64, s.Loads[2] > 32:
			out.Color(colors.Scheme("bad"))
		case s.Loads[0] > 32, s.Loads[2] > 16:
			out.Color(colors.Scheme("degraded"))
		}
		return out
	}).OnClick(startTaskManager)

	freeMem := meminfo.New().OutputFunc(func(m meminfo.Info) bar.Output {
		out := outputs.Pango(fontawesome.Icon("server"), spacer, m.Available().IEC())
		freeGigs := m.Available().In("GiB")
		switch {
		case freeGigs < 0.5:
			out.Urgent(true)
		case freeGigs < 1:
			out.Color(colors.Scheme("bad"))
		case freeGigs < 2:
			out.Color(colors.Scheme("degraded"))
		case freeGigs > 12:
			out.Color(colors.Scheme("good"))
		}
		return out
	}).OnClick(startTaskManager)

    batteryIndicator := battery.Default().OutputFunc(func(b battery.Info) bar.Output {
      multi := outputs.Multi().AddPango("remaining",
          fontawesome.Icon("battery", colors.Scheme("dim-icon")),
          b.RemainingPct(),
          "%")
      if b.PluggedIn() {
          multi = multi.AddPango("pluggedin",
              spacer,
              fontawesome.Icon("bolt", colors.Scheme("dim-icon")))
      }
      out := multi.Build()
      remaining := b.RemainingPct()
      if remaining < 10 {
        out.Urgent(true)
      }
      return out
    })

	g := group.Collapsing()

	panic(bar.Run(
		g.Add(freeMem),
		g.Add(loadAvg),
		g.Button(outputs.Text("+"), outputs.Text("-")),
        batteryIndicator,
		localtime,
	))
}
