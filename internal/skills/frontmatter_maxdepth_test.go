package skills

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type FrontmatterMaxDepthSuite struct {
	suite.Suite
}

func TestFrontmatterMaxDepthSuite(t *testing.T) {
	suite.Run(t, new(FrontmatterMaxDepthSuite))
}

func (s *FrontmatterMaxDepthSuite) TestMaxDepth() {
	scenarios := []struct {
		name    string
		content []byte
		want    int
	}{
		{
			name: "deve ler max_depth informado",
			content: []byte(`---
name: my-skill
version: 1.0.0
description: Uma skill com max_depth.
max_depth: 3
---
`),
			want: 3,
		},
		{
			name: "deve usar zero quando max_depth ausente",
			content: []byte(`---
name: my-skill
version: 1.0.0
description: Uma skill sem max_depth.
---
`),
			want: 0,
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			fm := NewCatalog().ParseFrontmatter(scenario.content)
			s.Equal(scenario.want, fm.MaxDepth, "MaxDepth")
		})
	}
}
