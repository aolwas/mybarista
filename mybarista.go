package main

import (
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/soumya92/barista"
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/modules/battery"
	"github.com/soumya92/barista/modules/clock"
	"github.com/soumya92/barista/modules/group"
	"github.com/soumya92/barista/modules/media"
	"github.com/soumya92/barista/modules/meminfo"
	"github.com/soumya92/barista/modules/sysinfo"
	"github.com/soumya92/barista/modules/weather"
	"github.com/soumya92/barista/modules/weather/openweathermap"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/pango"
)

var spacer = pango.Text(" ").XXSmall()

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

func mediaFormatFunc(m media.Info) bar.Output {
	if m.PlaybackStatus == media.Stopped || m.PlaybackStatus == media.Disconnected {
		return nil
	}
	artist := truncate(m.Artist, 20)
	title := truncate(m.Title, 40-len(artist))
	if len(title) < 20 {
		artist = truncate(m.Artist, 40-len(title))
	}
	iconAndPosition := pango.Text(" ")
	if m.PlaybackStatus == media.Playing {
		iconAndPosition.Append(
			spacer, pango.Textf("%s/%s",
				formatMediaTime(m.Position()),
				formatMediaTime(m.Length)),
		)
	}
	return outputs.Pango(iconAndPosition, spacer, title, " - ", artist)
}

func startTaskManager(e bar.Event) {
	if e.Button == bar.ButtonLeft {
		exec.Command("gnome-taskmanager").Run()
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

	colors.LoadFromMap(map[string]string{
		"good":     "#6d6",
		"degraded": "#dd6",
		"bad":      "#d66",
		"dim-icon": "#777",
	})

	localtime := clock.Local().
		Output(time.Second, func(now time.Time) bar.Output {
			return outputs.Pango(
				pango.Text(" "),
				now.Format("Jan 2 "),
				pango.Text(" "),
				now.Format("15:04:05"),
			)
		})
	localtime.OnClick(func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			exec.Command("gsimplecal").Run()
		}
	})

	// Weather information comes from OpenWeatherMap.
	// https://openweathermap.org/api.
	wthr := weather.New(
		openweathermap.Zipcode("31000", "FR").Build(),
	).Output(func(w weather.Weather) bar.Output {
		icon := ""
		switch w.Condition {
		case weather.Thunderstorm,
			weather.TropicalStorm,
			weather.Hurricane:
			icon = ""
		case weather.Drizzle,
			weather.Hail:
			icon = ""
		case weather.Rain:
			icon = ""
		case weather.Snow,
			weather.Sleet:
			icon = ""
		case weather.Mist,
			weather.Smoke,
			weather.Whirls,
			weather.Haze,
			weather.Fog:
			icon = ""
		case weather.Clear:
			if !w.Sunset.IsZero() && time.Now().After(w.Sunset) {
				icon = ""
			} else {
				icon = ""
			}
		case weather.PartlyCloudy:
			icon = ""
		case weather.Cloudy, weather.Overcast:
			icon = ""
		case weather.Tornado,
			weather.Windy:
			icon = ""
		}
		if icon == "" {
			icon = ""
		}
		return outputs.Pango(
			pango.Text(icon),
			pango.Textf(" %.1f℃", w.Temperature.Celsius()),
			pango.Textf(" (provided by %s)", w.Attribution).XSmall(),
		)
	})

	loadAvg := sysinfo.New().Output(func(s sysinfo.Info) bar.Output {
		out := outputs.Textf(" %0.2f %0.2f", s.Loads[0], s.Loads[2])
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
	})
	loadAvg.OnClick(startTaskManager)

	freeMem := meminfo.New().Output(func(m meminfo.Info) bar.Output {
		out := outputs.Pango(pango.Text(" "), outputs.IBytesize(m.Available()))
		freeGigs := m.Available().Gigabytes()
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
	})
	freeMem.OnClick(startTaskManager)

	batt := battery.Named("BAT0").Output(func(b battery.Info) bar.Output {
		var pstate *pango.Node
		if b.PluggedIn() {
			pstate = pango.Text(" ")
		} else {
			pstate = pango.Text("")
		}
		out := outputs.Pango(pstate, pango.Textf("%d%%", b.RemainingPct()))
		switch {
		case b.RemainingTime() < time.Duration(5)*time.Minute:
			out.Urgent(true)
		case b.RemainingTime() < time.Duration(10)*time.Minute:
			out.Color(colors.Scheme("bad"))
		case b.RemainingTime() < time.Duration(30)*time.Minute:
			out.Color(colors.Scheme("degraded"))
		case b.RemainingTime() > time.Duration(45)*time.Minute:
			out.Color(colors.Scheme("good"))
		}
		return out
	})

	gmplay := media.New("google-play-music-desktop-player").Output(mediaFormatFunc)

	g := group.Collapsing()

	panic(barista.Run(
		gmplay,
		g.Add(freeMem),
		g.Add(loadAvg),
		g.Button(outputs.Text("+"), outputs.Text("-")),
		wthr,
		batt,
		localtime,
	))
}
