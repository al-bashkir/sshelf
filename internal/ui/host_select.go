package ui

import (
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/charmbracelet/bubbles/list"

	"github.com/sahilm/fuzzy"
)

// hostSelectList provides common host selection, filtering, and list management
// logic shared by hostsModel, groupHostsModel, and hostPickerModel.
//
// It is embedded (not composed) in each model so that fields like allHosts,
// filtered, selected, and matchIdxes are promoted to the outer struct. The
// owning model wires l to its own list with bindList after construction.
type hostSelectList struct {
	l          *list.Model
	allHosts   []string
	filtered   []string
	selected   map[string]bool
	matchIdxes map[string][]int // fuzzy match indexes per host
	hiddenN    int              // hidden hosts among allHosts, set by filter
}

func newHostSelectList(hosts []string) hostSelectList {
	return hostSelectList{
		allHosts: append([]string(nil), hosts...),
		filtered: append([]string(nil), hosts...),
		selected: make(map[string]bool),
	}
}

// bindList attaches the owning model's list. Must be called once, after the
// model struct is allocated, before any other method.
func (s *hostSelectList) bindList(l *list.Model) { s.l = l }

// setListItems rebuilds list items from the current filtered hosts.
// markHidden sets the display-only hidden flag.
func (s *hostSelectList) setListItems(idx hostIndex, markHidden bool) {
	l := s.l
	items := make([]list.Item, 0, len(s.filtered))
	for _, h := range s.filtered {
		_, hasCfg := idx.lookup(h)
		hidden := markHidden && idx.isHidden(h)
		items = append(items, hostRow{
			host:           h,
			selected:       s.selected[h],
			hasCfg:         hasCfg,
			hidden:         hidden,
			matchedIndexes: s.matchIdxes[h],
		})
	}
	l.SetItems(items)
}

// refreshVisibleSelection updates the selected state on visible list items.
func (s *hostSelectList) refreshVisibleSelection() {
	l := s.l
	items := l.Items()
	for i := range items {
		row, ok := items[i].(hostRow)
		if !ok {
			continue
		}
		row.selected = s.selected[row.host]
		items[i] = row
	}
	l.SetItems(items)
}

// refreshVisibleBadges updates hasCfg badges on visible list items.
func (s *hostSelectList) refreshVisibleBadges(inv config.Inventory) {
	l := s.l
	idx := l.Index()
	items := l.Items()
	hidx := newHostIndex(inv)
	for i := range items {
		row, ok := items[i].(hostRow)
		if !ok {
			continue
		}
		_, ok = hidx.lookup(row.host)
		row.hasCfg = ok
		items[i] = row
	}
	l.SetItems(items)
	if idx >= 0 && idx < len(items) {
		l.Select(idx)
	}
}

// toggleCurrentSelection toggles selection on the currently focused host.
func (s *hostSelectList) toggleCurrentSelection() {
	row, ok := s.l.SelectedItem().(hostRow)
	if !ok || row.host == "" {
		return
	}
	if s.selected[row.host] {
		delete(s.selected, row.host)
	} else {
		s.selected[row.host] = true
	}
	s.refreshVisibleSelection()
}

// clearSelection deselects every host.
func (s *hostSelectList) clearSelection() {
	s.selected = make(map[string]bool)
	s.refreshVisibleSelection()
}

// selectAllFiltered selects every host currently passing the filter.
func (s *hostSelectList) selectAllFiltered() {
	for _, h := range s.filtered {
		s.selected[h] = true
	}
	s.refreshVisibleSelection()
}

// currentHost returns the focused host, or "" when the list is empty.
func (s *hostSelectList) currentHost() string {
	if row, ok := s.l.SelectedItem().(hostRow); ok {
		return row.host
	}
	return ""
}

// hostsToOpen returns the selected hosts, falling back to the focused one.
func (s *hostSelectList) hostsToOpen() []string {
	if sel := s.selectedHosts(); len(sel) > 0 {
		return sel
	}
	if h := s.currentHost(); h != "" {
		return []string{h}
	}
	return nil
}

// selectedHosts returns hosts that are currently selected, preserving
// the original allHosts order.
func (s *hostSelectList) selectedHosts() []string {
	if len(s.selected) == 0 {
		return nil
	}
	out := make([]string, 0, len(s.selected))
	for _, h := range s.allHosts {
		if s.selected[h] {
			out = append(out, h)
		}
	}
	return out
}

// filter fuzzy-filters hosts, optionally excludes entries, and preserves
// cursor position. excludeHidden drops hidden hosts from the result; markHidden
// flags them for display and counts them in hiddenN.
func (s *hostSelectList) filter(query string, inv config.Inventory, excludeHidden, markHidden bool) {
	prevHost := s.currentHost()
	idx := newHostIndex(inv)

	query = strings.TrimSpace(query)
	s.matchIdxes = nil
	if query == "" {
		s.filtered = append([]string(nil), s.allHosts...)
	} else {
		matches := fuzzy.Find(query, s.allHosts)
		s.filtered = make([]string, 0, len(matches))
		s.matchIdxes = make(map[string][]int, len(matches))
		for _, match := range matches {
			s.filtered = append(s.filtered, match.Str)
			if len(match.MatchedIndexes) > 0 {
				s.matchIdxes[match.Str] = match.MatchedIndexes
			}
		}
	}

	if excludeHidden {
		visible := make([]string, 0, len(s.filtered))
		for _, h := range s.filtered {
			if !idx.isHidden(h) {
				visible = append(visible, h)
			}
		}
		s.filtered = visible
	}

	s.hiddenN = 0
	if markHidden {
		for _, h := range s.allHosts {
			if idx.isHidden(h) {
				s.hiddenN++
			}
		}
	}

	s.setListItems(idx, markHidden)
	restoreCursor(s.l, s.allHosts, s.filtered, prevHost)
}

// restoreCursor re-selects prev after a re-filter. When prev is gone it picks
// the nearest surviving neighbour in `all` order — first looking forward, then
// backward — and otherwise falls back to the first row.
func restoreCursor(l *list.Model, all, visible []string, prev string) {
	if len(visible) == 0 {
		return
	}
	if prev != "" {
		pos := make(map[string]int, len(visible))
		for i, k := range visible {
			pos[k] = i
		}
		if i, ok := pos[prev]; ok {
			l.Select(i)
			return
		}
		prevIdx := -1
		for i, k := range all {
			if k == prev {
				prevIdx = i
				break
			}
		}
		if prevIdx >= 0 {
			for j := prevIdx + 1; j < len(all); j++ {
				if i, ok := pos[all[j]]; ok {
					l.Select(i)
					return
				}
			}
			for j := prevIdx - 1; j >= 0; j-- {
				if i, ok := pos[all[j]]; ok {
					l.Select(i)
					return
				}
			}
		}
	}
	l.Select(0)
}
