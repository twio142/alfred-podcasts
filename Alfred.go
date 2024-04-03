package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type Mod struct {
	Valid    bool   `json:"valid"`
	Subtitle string `json:"subtitle,omitempty"`
	Arg      string `json:"arg,omitempty"`
	Icon     struct {
		Path string `json:"path"`
	} `json:"icon,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

func (m *Mod) SetVar(name string, value string) {
	if m.Variables == nil {
		m.Variables = make(map[string]string)
	}
	m.Variables[name] = value
}

type Item struct {
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle,omitempty"`
	Arg          string `json:"arg,omitempty"`
	Valid        bool   `json:"valid"`
	AutoComplete string `json:"autocomplete,omitempty"`
	Type         string `json:"type,omitempty"`
	Match        string `json:"match,omitempty"`
	Text         struct {
		Copy      string `json:"copy,omitempty"`
		LargeType string `json:"largetype,omitempty"`
	} `json:"text,omitempty"`
	QuickLookURL string `json:"quicklookurl,omitempty"`
	Icon         struct {
		Path string `json:"path"`
	} `json:"icon,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
	Mods      struct {
		Cmd      Mod `json:"cmd"`
		Alt      Mod `json:"alt"`
		Shift    Mod `json:"shift"`
		Ctrl     Mod `json:"ctrl"`
		AltShift Mod `json:"alt+shift"`
	} `json:"mods,omitempty"`
	SortKey interface{} `json:"-"`
}

func (i *Item) SetVar(name string, value string) {
	if i.Variables == nil {
		i.Variables = make(map[string]string)
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

func (w *Workflow) WarnEmpty(s ...string) {
	var title = "No Result Found"
	if len(s) > 0 && s[0] != "" {
		title = s[0]
	}
	var icon = "../../resources/AlertCautionIcon.icns"
	if len(s) > 1 && s[1] != "" {
		icon = s[1]
	}
	w.Items = []Item{
		{
			Title: title,
			Valid: false,
			Icon: struct {
				Path string `json:"path"`
			}{Path: icon},
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
