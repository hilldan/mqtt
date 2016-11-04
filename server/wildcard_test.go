package server

import (
	"fmt"
	"testing"
)

func TestMatchPath(t *testing.T) {
	var ts = []struct {
		sub1   string
		sub2   string
		result bool
	}{
		{"sport/tennis/player1/#", "sport/tennis/player1", true},
		{"sport/tennis/player1/#", "sport/tennis/player1/ranking", true},
		{"sport/tennis/player1/#", "sport/tennis/player1/score/wimbledon", true},
		{"sport/#", "sport", true},
		{"#", "sport/tennis/player1/score/wimbledon", true},
		{"+/tennis/#", "sport/tennis/player1/score/wimbledon", true},

		{"sport/tennis/+/ranking", "sport/tennis/xxx/ranking", true},
		{"sport/tennis/+", "sport/tennis/player1", true},
		{"sport/tennis/+", "sport/tennis/player2", true},
		{"sport/tennis/+", "sport/tennis/", true},
		{"sport/tennis/+", "sport/tennis/player1/ranking", false},
		{"sport/+", "sport", false},
		{"sport/+", "sport/", true},
		{"+/+", "/finance", true},
		{"/+", "/finance", true},
		{"+", "/finance", false},
		{"+", "finance", true},

		{"+/monitor/Clients", "$SYS/monitor/Clients", false},
		{"$SYS/#", "$SYS/", true},
		{"$SYS/monitor/+", "$SYS/monitor/Clients", true},
	}

	for _, v := range ts {
		valid, p1 := check(v.sub1)
		if !valid {
			t.Errorf("subject is invalid: %s", v.sub1)
		}
		valid, p2 := check(v.sub2)
		if !valid {
			t.Errorf("subject is invalid: %s", v.sub2)
		}
		matched := matchPath(p1, p2)
		if matched != v.result {
			t.Errorf("'%s' and '%s' should match %v", v.sub1, v.sub2, v.result)
		}

		matched = matchOne(v.sub1, v.sub2)
		if matched != v.result {
			t.Errorf("'%s' and '%s' should match %v", v.sub1, v.sub2, v.result)
		}

		flag, relate := compare(p1, p2)
		if relate != v.result {
			t.Errorf("relate want %b actual %b ", v.result, relate)
		}

		flag, relate = compare(p2, p1)
		if relate != v.result {
			t.Errorf("relate want %b actual %b ", v.result, relate)
		}

		flag, relate = compare(p1, p1[:len(p1)-1])
		if relate != true && flag != 0 {
			t.Errorf("relate want true actual %v ", relate)
		}

		fmt.Println(flag, relate)
	}
}
