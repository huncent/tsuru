// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"

	"gopkg.in/check.v1"
	"gopkg.in/mgo.v2/bson"
)

func (s *S) createService() {
	s.service = &Service{Name: "my_service"}
	s.service.Create()
}

func (s *S) TestGetService(c *check.C) {
	s.createService()
	anotherService := Service{Name: s.service.Name}
	anotherService.Get()
	c.Assert(anotherService.Name, check.Equals, s.service.Name)
}

func (s *S) TestGetServiceReturnsErrorIfTheServiceIsDeleted(c *check.C) {
	se := Service{Name: "anything"}
	err := se.Get()
	c.Assert(err, check.NotNil)
}

func (s *S) TestCreateService(c *check.C) {
	endpt := map[string]string{
		"production": "somehost.com",
	}
	service := &Service{
		Name:       "my_service",
		Username:   "test",
		Endpoint:   endpt,
		OwnerTeams: []string{s.team.Name},
	}
	err := service.Create()
	c.Assert(err, check.IsNil)
	se := Service{Name: service.Name}
	err = se.Get()
	c.Assert(err, check.IsNil)
	c.Assert(se.Name, check.Equals, service.Name)
	c.Assert(se.Endpoint["production"], check.Equals, endpt["production"])
	c.Assert(se.OwnerTeams, check.DeepEquals, []string{s.team.Name})
	c.Assert(se.IsRestricted, check.Equals, false)
	c.Assert(se.Username, check.Equals, "test")
}

func (s *S) TestDeleteService(c *check.C) {
	s.createService()
	err := s.service.Delete()
	defer s.conn.Services().Remove(bson.M{"_id": s.service.Name})
	c.Assert(err, check.IsNil)
	l, err := s.conn.Services().Find(bson.M{"_id": s.service.Name}).Count()
	c.Assert(err, check.IsNil)
	c.Assert(l, check.Equals, 0)
}

func (s *S) TestGetClient(c *check.C) {
	endpoints := map[string]string{
		"production": "http://mysql.api.com",
	}
	service := Service{Name: "redis", Password: "abcde", Endpoint: endpoints}
	cli, err := service.getClient("production")
	expected := &Client{
		serviceName: "redis",
		endpoint:    endpoints["production"],
		username:    "redis",
		password:    "abcde",
	}
	c.Assert(err, check.IsNil)
	c.Assert(cli, check.DeepEquals, expected)
}

func (s *S) TestGetClientWithServiceUsername(c *check.C) {
	endpoints := map[string]string{
		"production": "http://mysql.api.com",
	}
	service := Service{Name: "redis", Username: "redis_test", Password: "abcde", Endpoint: endpoints}
	cli, err := service.getClient("production")
	expected := &Client{
		serviceName: "redis",
		endpoint:    endpoints["production"],
		username:    "redis_test",
		password:    "abcde",
	}
	c.Assert(err, check.IsNil)
	c.Assert(cli, check.DeepEquals, expected)
}

func (s *S) TestGetClientWithouHTTP(c *check.C) {
	endpoints := map[string]string{
		"production": "mysql.api.com",
	}
	service := Service{Name: "redis", Endpoint: endpoints}
	cli, err := service.getClient("production")
	c.Assert(err, check.IsNil)
	c.Assert(cli.endpoint, check.Equals, "http://mysql.api.com")
}

func (s *S) TestGetClientWithHTTPS(c *check.C) {
	endpoints := map[string]string{
		"production": "https://mysql.api.com",
	}
	service := Service{Name: "redis", Endpoint: endpoints}
	cli, err := service.getClient("production")
	c.Assert(err, check.IsNil)
	c.Assert(cli.endpoint, check.Equals, "https://mysql.api.com")
}

func (s *S) TestGetClientWithUnknownEndpoint(c *check.C) {
	endpoints := map[string]string{
		"production": "http://mysql.api.com",
	}
	service := Service{Name: "redis", Endpoint: endpoints}
	cli, err := service.getClient("staging")
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^Unknown endpoint: staging$")
	c.Assert(cli, check.IsNil)
}

func (s *S) TestGetUsername(c *check.C) {
	service := Service{Name: "test"}
	c.Assert(service.Name, check.Equals, service.GetUsername())
	service.Username = "test_test"
	c.Assert(service.Username, check.Equals, service.GetUsername())
}

func (s *S) TestGrantAccessShouldAddTeamToTheService(c *check.C) {
	s.createService()
	err := s.service.GrantAccess(s.team)
	c.Assert(err, check.IsNil)
	c.Assert(*s.team, HasAccessTo, *s.service)
}

func (s *S) TestGrantAccessShouldReturnErrorIfTheTeamAlreadyHasAcessToTheService(c *check.C) {
	s.createService()
	err := s.service.GrantAccess(s.team)
	c.Assert(err, check.IsNil)
	err = s.service.GrantAccess(s.team)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^This team already has access to this service$")
}

func (s *S) TestRevokeAccessShouldRemoveTeamFromService(c *check.C) {
	s.createService()
	err := s.service.GrantAccess(s.team)
	c.Assert(err, check.IsNil)
	err = s.service.RevokeAccess(s.team)
	c.Assert(err, check.IsNil)
	c.Assert(*s.team, check.Not(HasAccessTo), *s.service)
}

func (s *S) TestRevokeAcessShouldReturnErrorIfTheTeamDoesNotHaveAccessToTheService(c *check.C) {
	s.createService()
	err := s.service.RevokeAccess(s.team)
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "^This team does not have access to this service$")
}

func (s *S) TestGetServicesNames(c *check.C) {
	s1 := Service{Name: "Foo"}
	s2 := Service{Name: "Bar"}
	s3 := Service{Name: "FooBar"}
	sNames := GetServicesNames([]Service{s1, s2, s3})
	c.Assert(sNames, check.DeepEquals, []string{"Foo", "Bar", "FooBar"})
}

func (s *S) TestUpdateService(c *check.C) {
	service := Service{Name: "something"}
	err := service.Create()
	c.Assert(err, check.IsNil)
	defer s.conn.Services().Remove(bson.M{"_id": service.Name})
	service.Doc = "doc"
	err = service.Update()
	c.Assert(err, check.IsNil)
	err = s.conn.Services().Find(bson.M{"_id": service.Name}).One(&service)
	c.Assert(err, check.IsNil)
	c.Assert(service.Doc, check.Equals, "doc")
}

func (s *S) TestUpdateServiceReturnErrorIfServiceDoesNotExist(c *check.C) {
	service := Service{Name: "something"}
	err := service.Update()
	c.Assert(err, check.NotNil)
}

func (s *S) TestGetServicesByOwnerTeamsAndServices(c *check.C) {
	srvc := Service{Name: "mongodb", OwnerTeams: []string{s.team.Name}, Endpoint: map[string]string{}, Teams: []string{}}
	err := srvc.Create()
	c.Assert(err, check.IsNil)
	defer srvc.Delete()
	srvc2 := Service{Name: "mysql", Teams: []string{s.team.Name}}
	err = srvc2.Create()
	c.Assert(err, check.IsNil)
	defer srvc2.Delete()
	services, err := GetServicesByOwnerTeamsAndServices([]string{s.team.Name}, nil)
	c.Assert(err, check.IsNil)
	expected := []Service{srvc}
	c.Assert(services, check.DeepEquals, expected)
}

func (s *S) TestGetServicesByOwnerTeamsAndServicesWithServices(c *check.C) {
	srvc := Service{Name: "mongodb", OwnerTeams: []string{s.team.Name}, Endpoint: map[string]string{}, Teams: []string{}}
	err := srvc.Create()
	c.Assert(err, check.IsNil)
	srvc2 := Service{Name: "mysql", Teams: []string{s.team.Name}}
	err = srvc2.Create()
	c.Assert(err, check.IsNil)
	services, err := GetServicesByOwnerTeamsAndServices([]string{s.team.Name}, []string{srvc2.Name})
	c.Assert(err, check.IsNil)
	c.Assert(services, check.HasLen, 2)
	var names []string
	for _, s := range services {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	c.Assert(names, check.DeepEquals, []string{"mongodb", "mysql"})
}

func (s *S) TestGetServicesByOwnerTeamsAndServicesShouldNotReturnsDeletedServices(c *check.C) {
	service := Service{Name: "mysql", OwnerTeams: []string{s.team.Name}, Endpoint: map[string]string{}, Teams: []string{}}
	err := service.Create()
	c.Assert(err, check.IsNil)
	deletedService := Service{Name: "mongodb", OwnerTeams: []string{s.team.Name}}
	err = deletedService.Create()
	c.Assert(err, check.IsNil)
	err = deletedService.Delete()
	c.Assert(err, check.IsNil)
	services, err := GetServicesByOwnerTeamsAndServices([]string{s.team.Name}, nil)
	c.Assert(err, check.IsNil)
	c.Assert(err, check.IsNil)
	expected := []Service{service}
	c.Assert(services, check.DeepEquals, expected)
}

func (s *S) TestServiceModelMarshalJSON(c *check.C) {
	sm := []ServiceModel{
		{Service: "mysql"},
		{Service: "mongo"},
	}
	data, err := json.Marshal(&sm)
	c.Assert(err, check.IsNil)
	expected := make([]map[string]interface{}, 2)
	expected[0] = map[string]interface{}{
		"service":   "mysql",
		"instances": nil,
		"plans":     nil,
	}
	expected[1] = map[string]interface{}{
		"service":   "mongo",
		"instances": nil,
		"plans":     nil,
	}
	result := make([]map[string]interface{}, 2)
	err = json.Unmarshal(data, &result)
	c.Assert(err, check.IsNil)
	c.Assert(result, check.DeepEquals, expected)
}

func (s *S) TestProxy(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	service := Service{
		Name:     "mongodb",
		Endpoint: map[string]string{"production": ts.URL},
	}
	err := s.conn.Services().Insert(service)
	c.Assert(err, check.IsNil)
	defer s.conn.Services().RemoveId(service.Name)
	request, err := http.NewRequest("DELETE", "/something", nil)
	c.Assert(err, check.IsNil)
	recorder := httptest.NewRecorder()
	err = Proxy(&service, "/aaa", recorder, request)
	c.Assert(err, check.IsNil)
	c.Assert(recorder.Code, check.Equals, http.StatusNoContent)
}
