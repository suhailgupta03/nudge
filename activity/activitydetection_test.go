package activity

import (
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"log"
	prp "nudge/internal/database/pr"
	"os"
	"testing"
	"time"
)

type checkForActivityTestCase struct {
	Name                   string
	PRModel                prp.PRModel
	Config                 map[string]interface{}
	ExpectedActivityResult bool
}

func TestCheckForActivity(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.Lshortfile)
	k := koanf.New(".")

	testCases := []checkForActivityTestCase{
		{
			Name: "Activity detected - hours",
			PRModel: prp.PRModel{
				WorkflowLastActivity: func() *int64 {
					t := time.Now().Unix()
					return &t
				}(),
			},
			Config: map[string]interface{}{
				"bot.interval_to_wait.unit": "h",
				"bot.interval_to_wait.time": 24.0,
			},
			ExpectedActivityResult: true,
		},
		{
			Name: "Activity not detected - hours",
			PRModel: prp.PRModel{
				WorkflowLastActivity: func() *int64 {
					t := time.Now().Add(-25 * time.Hour).Unix()
					return &t
				}(),
			},
			Config: map[string]interface{}{
				"bot.interval_to_wait.unit": "h",
				"bot.interval_to_wait.time": 24.0,
			},
			ExpectedActivityResult: false,
		},
		{
			Name: "Activity detected - minutes",
			PRModel: prp.PRModel{
				WorkflowLastActivity: func() *int64 {
					t := time.Now().Unix()
					return &t
				}(),
			},
			Config: map[string]interface{}{
				"bot.interval_to_wait.unit": "m",
				"bot.interval_to_wait.time": 1440.0, // 24 * 60 minutes
			},
			ExpectedActivityResult: true,
		},
		{
			Name: "Activity not detected - minutes",
			PRModel: prp.PRModel{
				WorkflowLastActivity: func() *int64 {
					t := time.Now().Add(-1500 * time.Minute).Unix() // 25 hours in minutes
					return &t
				}(),
			},
			Config: map[string]interface{}{
				"bot.interval_to_wait.unit": "m",
				"bot.interval_to_wait.time": 1440.0, // 24 * 60 minutes
			},
			ExpectedActivityResult: false,
		},
		{
			Name: "Nil WorkflowLastActivity - activity not detected",
			PRModel: prp.PRModel{
				WorkflowLastActivity: nil,
			},
			Config: map[string]interface{}{
				"bot.interval_to_wait.unit": "h",
				"bot.interval_to_wait.time": 24.0,
			},
			ExpectedActivityResult: false,
		},
	}

	// Iterate through the test cases
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Load the configuration for the test case
			k.Load(confmap.Provider(testCase.Config, "."), nil)

			// Create a dummy Activity instance
			activity := &Activity{
				ko: k,
				lo: logger,
			}

			// Run the checkForActivity function
			result := activity.CheckForActivity(testCase.PRModel)

			// Check the result
			assert.Equal(t, testCase.ExpectedActivityResult, result.Detected)
		})
	}
}

type CheckForActivityMock struct {
	mock.Mock
}

func (m *CheckForActivityMock) CheckForActivity(prModel prp.PRModel) *ActivityDetection {
	args := m.Called(prModel)
	return args.Get(0).(*ActivityDetection)
}

type isPRMovingTestCase struct {
	Name                         string
	OpenPR                       prp.PRModel
	MockedCheckForActivityResult bool
	ExpectedResult               bool
	WithinLifeTime               bool
}

func TestIsPRMoving(t *testing.T) {
	ko := koanf.New(".")
	lo := log.New(os.Stdout, "", log.LstdFlags)

	testCases := []isPRMovingTestCase{
		{
			Name: "Within lifetime (moving)",
			OpenPR: prp.PRModel{
				PRCreatedAt: time.Now().Add(-23 * time.Hour).Unix(),
				LifeTime:    24,
			},
			MockedCheckForActivityResult: true,
			ExpectedResult:               true,
			WithinLifeTime:               true,
		},
		{
			Name: "Beyond lifetime but with activity (moving)",
			OpenPR: prp.PRModel{
				PRCreatedAt:          time.Now().Add(-25 * time.Hour).Unix(),
				LifeTime:             24,
				WorkflowLastActivity: int64Ptr(time.Now().Add(-2 * time.Hour).Unix()),
			},
			MockedCheckForActivityResult: true,
			ExpectedResult:               true,
			WithinLifeTime:               false,
		},
		{
			Name: "Beyond lifetime without activity (not moving)",
			OpenPR: prp.PRModel{
				PRCreatedAt:          time.Now().Add(-26 * time.Hour).Unix(),
				LifeTime:             24,
				WorkflowLastActivity: int64Ptr(time.Now().Add(-28 * time.Hour).Unix()),
			},
			MockedCheckForActivityResult: false,
			ExpectedResult:               false,
			WithinLifeTime:               false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			activity := Init(ko, nil, lo)

			checkForActivityMock := &CheckForActivityMock{}
			checkForActivityMock.On("CheckForActivity", testCase.OpenPR).Return(&ActivityDetection{Detected: testCase.MockedCheckForActivityResult})

			moving := activity.IsPRMoving(testCase.OpenPR, checkForActivityMock)
			assert.Equal(t, testCase.ExpectedResult, *moving)
			if !testCase.WithinLifeTime {
				checkForActivityMock.AssertCalled(t, "CheckForActivity", testCase.OpenPR)
			}
		})
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}
