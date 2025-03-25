package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Icon struct {
	Path string `json:"path"`
}

type Mod struct {
	Valid     bool           `json:"valid,omitempty"`
	Subtitle  string         `json:"subtitle,omitempty"`
	Arg       string         `json:"arg,omitempty"`
	Icon      *Icon          `json:"icon,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
}

func (m *Mod) SetVar(name string, value any) {
	if m.Variables == nil {
		m.Variables = make(map[string]any)
	}
	m.Variables[name] = value
}

type Item struct {
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle,omitempty"`
	Arg          string `json:"arg,omitempty"`
	Valid        *bool  `json:"valid,omitempty"`
	AutoComplete string `json:"autocomplete,omitempty"`
	UID          string `json:"uid,omitempty"`
	Type         string `json:"type,omitempty"`
	Match        string `json:"match,omitempty"`
	Text         struct {
		Copy      string `json:"copy,omitempty"`
		LargeType string `json:"largetype,omitempty"`
	} `json:"text"`
	QuickLookURL string         `json:"quicklookurl,omitempty"`
	Icon         *Icon          `json:"icon,omitempty"`
	Variables    map[string]any `json:"variables,omitempty"`
	Mods         struct {
		Cmd       *Mod `json:"cmd,omitempty"`
		Alt       *Mod `json:"alt,omitempty"`
		Shift     *Mod `json:"shift,omitempty"`
		Ctrl      *Mod `json:"ctrl,omitempty"`
		Fn        *Mod `json:"fn,omitempty"`
		AltShift  *Mod `json:"alt+shift,omitempty"`
		CtrlShift *Mod `json:"ctrl+shift,omitempty"`
		CmdShift  *Mod `json:"cmd+shift,omitempty"`
	} `json:"mods"`
}

func (i *Item) SetVar(name string, value any) {
	if i.Variables == nil {
		i.Variables = make(map[string]any)
	}
	i.Variables[name] = value
}

type Workflow struct {
	Items     []Item            `json:"items,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

func (w *Workflow) AddItem(item *Item) {
	w.Items = append(w.Items, *item)
}

func (w *Workflow) UnshiftItem(item *Item) {
	w.Items = append([]Item{*item}, w.Items...)
}

func (w *Workflow) WarnEmpty(s ...string) {
	title := "No Result Found"
	if len(s) > 0 && s[0] != "" {
		title = s[0]
	}
	icon := os.Getenv("alfred_preferences") + "/resources/AlertCautionIcon.icns"
	if len(s) > 1 && s[1] != "" {
		icon = s[1]
	}
	valid := false
	w.Items = []Item{
		{
			Title: title,
			Valid: &valid,
			Icon:  &Icon{Path: icon},
		},
	}
}

func (w *Workflow) SetVar(name string, value string) {
	if w.Variables == nil {
		w.Variables = make(map[string]string)
	}
	w.Variables[name] = value
}

func (w *Workflow) Output() {
	if len(w.Items) == 0 {
		return
	}
	jsonItems, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		log.Println("Error:", err)
		return
	}
	fmt.Println(string(jsonItems))
}
