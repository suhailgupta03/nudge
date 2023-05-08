package buflog

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	maxLines := 10
	bufLog := New(maxLines)

	assert.Equal(t, maxLines, bufLog.maxLines)
	assert.NotNil(t, bufLog.buf)
	assert.NotNil(t, bufLog.lines)
}

func TestWrite(t *testing.T) {
	t.Run("write_less_than_max_lines", func(t *testing.T) {
		maxLines := 3
		bufLog := New(maxLines)

		lines := []string{"Hello", "World", "BufLog"}
		for _, line := range lines {
			_, _ = io.WriteString(bufLog, line+"\n")
		}

		assert.Equal(t, lines, bufLog.Lines())
	})

	t.Run("write_more_than_max_lines", func(t *testing.T) {
		maxLines := 2
		bufLog := New(maxLines)

		lines := []string{"First", "Second", "Third", "Fourth"}
		expectedLines := []string{"Third", "Fourth"}

		for _, line := range lines {
			_, _ = io.WriteString(bufLog, line+"\n")
		}

		assert.Equal(t, expectedLines, bufLog.Lines())
	})
}

func TestLines(t *testing.T) {
	maxLines := 3
	bufLog := New(maxLines)

	lines := []string{"line1", "line2", "line3"}
	for _, line := range lines {
		_, _ = io.WriteString(bufLog, line+"\n")
	}

	out := bufLog.Lines()

	assert.Equal(t, lines, out)
}
