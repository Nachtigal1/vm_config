package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kre-college/lms/pkg/inventory/service"
	"github.com/kre-college/lms/pkg/models"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	mockServices "github.com/kre-college/lms/pkg/inventory/service/mocks"
	jwt "github.com/kre-college/lms/pkg/jwt"
)

var testModel = &models.Room{
	ID:        1,
	Number:    "10-Test-A",
	Type:      models.TypeClassRoom,
	Building:  "Building 123",
	Floor:     0,
	Seats:     30,
	Computers: 15,
}
var testID = 1
var testIDString = "1"

var testModelArray = []*models.Room{testModel}
var filledModelArrayBytes = marshalFunc(testModelArray)

var testToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFeHBpcmVzQXQiOjE2NjkwMjU2ODgsIkZ1bGxVc2VyTmFtZSI6IkFkbWluIEFkbWluIiwiVXNlcklEIjoxMDAwMDAwfQ.IGGXmVRyDz561Q6BX-XiH0pWrVOkhzav4SifD80HQH0"
var claims, _ = jwt.ExtractClaims(testToken)

var path = "http://localhost:8093/rooms"

func TestNewFetchRoomsHandler(t *testing.T) {
	type mockBehavior func(ctx context.Context, academicYearID string, s *mockServices.MockRoomSvc)
	testTable := []struct {
		name                string
		inputBody           string
		academicYearID      string
		mockBehavior        mockBehavior
		expectedStatusCode  int
		expectedRequestBody string
	}{
		{
			name:           "OK",
			inputBody:      "",
			academicYearID: "",
			mockBehavior: func(ctx context.Context, academicYearID string, s *mockServices.MockRoomSvc) {
				s.EXPECT().FetchRooms(ctx, academicYearID).Return(testModelArray, nil)
			},
			expectedStatusCode:  200,
			expectedRequestBody: string(filledModelArrayBytes),
		},
		{
			name:           "BadRequest",
			inputBody:      "",
			academicYearID: "",
			mockBehavior: func(ctx context.Context, academicYearID string, s *mockServices.MockRoomSvc) {
				s.EXPECT().FetchRooms(ctx, academicYearID).Return(nil, service.ErrConvID)
			},
			expectedStatusCode:  400,
			expectedRequestBody: `{"code":400,"message":"converting id error"}`,
		},
		{
			name:           "ErrInternal",
			inputBody:      "",
			academicYearID: "",
			mockBehavior: func(ctx context.Context, academicYearID string, s *mockServices.MockRoomSvc) {
				s.EXPECT().FetchRooms(ctx, academicYearID).Return(nil, errors.New("unknown error"))
			},
			expectedStatusCode:  500,
			expectedRequestBody: `{"code":500,"message":"internal server error"}`,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			vars := map[string]string{
				"academic_year_id": fmt.Sprint(testCase.academicYearID),
			}
			req = mux.SetURLVars(req, vars)

			svc := mockServices.NewMockRoomSvc(c)
			handler := NewFetchRoomsHandler(svc)
			testCase.mockBehavior(req.Context(), testCase.academicYearID, svc)

			handler.ServeHTTP(w, req)

			assert.Equal(t, testCase.expectedStatusCode, w.Code)
			assert.Equal(t, testCase.expectedRequestBody, w.Body.String())
		})
	}
}

func TestNewAddRoomHandler(t *testing.T) {
	type mockBehavior func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc)
	testTable := []struct {
		name                string
		inputBody           []*models.Room
		inputJSON           []byte
		jwtToken            string
		claims              *jwt.Claims
		mockBehavior        mockBehavior
		expectedStatusCode  int
		expectedRequestBody string
	}{
		{
			name:      "OK",
			inputBody: testModelArray,
			inputJSON: filledModelArrayBytes,
			jwtToken:  testToken,
			claims:    claims,
			mockBehavior: func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc) {
				s.EXPECT().AddRooms(ctx, claims, room).Return(nil)
			},
			expectedStatusCode:  200,
			expectedRequestBody: string(filledModelArrayBytes),
		},
		{
			name:                "UnmarshalError",
			inputBody:           nil,
			inputJSON:           []byte(`garbage`),
			jwtToken:            testToken,
			claims:              claims,
			mockBehavior:        func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc) {},
			expectedStatusCode:  500,
			expectedRequestBody: `{"code":500,"message":"internal server error"}`,
		},
		{
			name:                "BadJwt",
			inputBody:           testModelArray,
			inputJSON:           filledModelArrayBytes,
			jwtToken:            "",
			claims:              nil,
			mockBehavior:        func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc) {},
			expectedStatusCode:  400,
			expectedRequestBody: `{"code":400,"message":"error bad request"}`,
		},
		{
			name:      "Conflict",
			inputBody: testModelArray,
			inputJSON: filledModelArrayBytes,
			jwtToken:  testToken,
			claims:    claims,
			mockBehavior: func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc) {
				s.EXPECT().AddRooms(ctx, claims, room).Return(service.ErrConflict)
			},
			expectedStatusCode:  409,
			expectedRequestBody: `{"code":409,"message":"error conflict"}`,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(testCase.inputJSON))
			req.Header.Add("Authorization", testCase.jwtToken)
			w := httptest.NewRecorder()

			svc := mockServices.NewMockRoomSvc(c)
			handler := NewAddRoomsHandler(svc)
			testCase.mockBehavior(req.Context(), testCase.claims, testCase.inputBody, svc)

			handler.ServeHTTP(w, req)

			assert.Equal(t, testCase.expectedStatusCode, w.Code)
			assert.Equal(t, testCase.expectedRequestBody, w.Body.String())
		})
	}
}

func TestNewUpdateRoomsHandler(t *testing.T) {
	type mockBehavior func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc)
	testTable := []struct {
		name                string
		inputBody           []*models.Room
		inputJSON           []byte
		jwtToken            string
		claims              *jwt.Claims
		mockBehavior        mockBehavior
		expectedStatusCode  int
		expectedRequestBody string
	}{
		{
			name:      "OK",
			inputBody: testModelArray,
			inputJSON: filledModelArrayBytes,
			jwtToken:  testToken,
			claims:    claims,
			mockBehavior: func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc) {
				s.EXPECT().UpdateRooms(ctx, claims, room).Return(nil)
			},
			expectedStatusCode:  200,
			expectedRequestBody: string(filledModelArrayBytes),
		},
		{
			name:                "UnmarshalError",
			inputBody:           nil,
			inputJSON:           []byte(`garbage`),
			jwtToken:            testToken,
			claims:              claims,
			mockBehavior:        func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc) {},
			expectedStatusCode:  500,
			expectedRequestBody: `{"code":500,"message":"internal server error"}`,
		},
		{
			name:                "BadJwt",
			inputBody:           testModelArray,
			inputJSON:           filledModelArrayBytes,
			jwtToken:            "",
			claims:              nil,
			mockBehavior:        func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc) {},
			expectedStatusCode:  400,
			expectedRequestBody: `{"code":400,"message":"error bad request"}`,
		},
		{
			name:      "NotFound",
			inputBody: testModelArray,
			inputJSON: filledModelArrayBytes,
			jwtToken:  testToken,
			claims:    claims,
			mockBehavior: func(ctx context.Context, claims *jwt.Claims, room []*models.Room, s *mockServices.MockRoomSvc) {
				s.EXPECT().UpdateRooms(ctx, claims, room).Return(service.ErrNoRecords)
			},
			expectedStatusCode:  404,
			expectedRequestBody: `{"code":404,"message":"error no records"}`,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(testCase.inputJSON))
			req.Header.Add("Authorization", testCase.jwtToken)
			w := httptest.NewRecorder()

			svc := mockServices.NewMockRoomSvc(c)
			handler := NewUpdateRoomsHandler(svc)
			testCase.mockBehavior(req.Context(), testCase.claims, testCase.inputBody, svc)

			handler.ServeHTTP(w, req)

			assert.Equal(t, testCase.expectedStatusCode, w.Code)
			assert.Equal(t, testCase.expectedRequestBody, w.Body.String())
		})
	}
}

func TestNewDeleteRoomsHandler(t *testing.T) {
	type mockBehavior func(ctx context.Context, claims *jwt.Claims, id []int, s *mockServices.MockRoomSvc)
	testTable := []struct {
		name                string
		inputBody           string
		roomsID             []int
		jwtToken            string
		claims              *jwt.Claims
		mockBehavior        mockBehavior
		expectedStatusCode  int
		expectedRequestBody string
	}{
		{
			name:      "OK",
			inputBody: "",
			roomsID:   []int{testID},
			jwtToken:  testToken,
			claims:    claims,
			mockBehavior: func(ctx context.Context, claims *jwt.Claims, ids []int, s *mockServices.MockRoomSvc) {
				s.EXPECT().DeleteRooms(ctx, claims, ids).Return(nil)
			},
			expectedStatusCode:  200,
			expectedRequestBody: "",
		},
		{
			name:                "BadJwt",
			inputBody:           "",
			roomsID:             []int{testID},
			jwtToken:            "",
			claims:              nil,
			mockBehavior:        func(ctx context.Context, claims *jwt.Claims, id []int, s *mockServices.MockRoomSvc) {},
			expectedStatusCode:  400,
			expectedRequestBody: `{"code":400,"message":"error bad request"}`,
		},
		{
			name:      "ErrInternal",
			inputBody: "",
			roomsID:   []int{testID},
			jwtToken:  testToken,
			claims:    claims,
			mockBehavior: func(ctx context.Context, claims *jwt.Claims, id []int, s *mockServices.MockRoomSvc) {
				s.EXPECT().DeleteRooms(ctx, claims, id).Return(errors.New("unknown error"))
			},
			expectedStatusCode:  500,
			expectedRequestBody: `{"code":500,"message":"internal server error"}`,
		},
		{
			name:      "NoRecords",
			inputBody: "",
			roomsID:   []int{testID},
			jwtToken:  testToken,
			claims:    claims,
			mockBehavior: func(ctx context.Context, claims *jwt.Claims, id []int, s *mockServices.MockRoomSvc) {
				s.EXPECT().DeleteRooms(ctx, claims, id).Return(service.ErrNoRecords)
			},
			expectedStatusCode:  404,
			expectedRequestBody: `{"code":404,"message":"error no records"}`,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			req := httptest.NewRequest(http.MethodDelete, path, bytes.NewBuffer(marshalFunc(testCase.roomsID)))
			req.Header.Add("Authorization", testCase.jwtToken)
			w := httptest.NewRecorder()

			svc := mockServices.NewMockRoomSvc(c)
			handler := NewDeleteRoomsByIDsHandler(svc)
			testCase.mockBehavior(req.Context(), testCase.claims, testCase.roomsID, svc)

			handler.ServeHTTP(w, req)

			assert.Equal(t, testCase.expectedStatusCode, w.Code)
			assert.Equal(t, testCase.expectedRequestBody, w.Body.String())
		})
	}
}

func TestNewFetchRoomHistoryHandler(t *testing.T) {
	type mockBehavior func(ctx context.Context, id int, s *mockServices.MockRoomSvc)
	testTable := []struct {
		name                string
		roomId              string
		mockBehavior        mockBehavior
		expectedStatusCode  int
		expectedRequestBody string
	}{
		{
			name:   "OK",
			roomId: testIDString,
			mockBehavior: func(ctx context.Context, id int, s *mockServices.MockRoomSvc) {
				s.EXPECT().FetchRoomHistory(ctx, id).Return(testModelArray, nil)
			},
			expectedStatusCode:  200,
			expectedRequestBody: string(filledModelArrayBytes),
		},
		{
			name:                "BadRequest",
			mockBehavior:        func(ctx context.Context, id int, s *mockServices.MockRoomSvc) {},
			expectedStatusCode:  400,
			expectedRequestBody: `{"code":400,"message":"converting id error"}`,
		},
		{
			name:   "ErrInternal",
			roomId: testIDString,
			mockBehavior: func(ctx context.Context, id int, s *mockServices.MockRoomSvc) {
				s.EXPECT().FetchRoomHistory(ctx, id).Return(nil, errors.New("unknown error"))
			},
			expectedStatusCode:  500,
			expectedRequestBody: `{"code":500,"message":"internal server error"}`,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			vars := map[string]string{
				"roomId": fmt.Sprint(testCase.roomId),
			}
			req = mux.SetURLVars(req, vars)

			svc := mockServices.NewMockRoomSvc(c)
			handler := NewFetchRoomHistoryHandler(svc)
			testCase.mockBehavior(req.Context(), testID, svc)

			handler.ServeHTTP(w, req)

			assert.Equal(t, testCase.expectedStatusCode, w.Code)
			assert.Equal(t, testCase.expectedRequestBody, w.Body.String())
		})
	}
}

func marshalFunc(input interface{}) []byte {
	var testJSON, _ = json.Marshal(&input)
	return testJSON
}

