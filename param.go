package main

import "strings"

type param []string

func (p param) MarshalJSON() ([]byte, error) {
	if len(p) == 1 {
		b := make([]byte, 0, len(p[0])+2)
		b = append(b, '"')
		b = append(b, []byte(strings.ReplaceAll(p[0], `"`, `\"`))...)
		b = append(b, '"')
		return b, nil
	}

	b := make([]byte, 0, 8)
	b = append(b, '[')
	for i, v := range p {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '"')
		b = append(b, []byte(strings.ReplaceAll(v, `"`, `\"`))...)
		b = append(b, '"')
	}
	b = append(b, ']')

	return b, nil
}

func (p param) String() string {
	if len(p) == 1 {
		return p[0]
	}
	return strings.Join(p, ",")
}
