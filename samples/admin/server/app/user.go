/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/config"
)

func init() {
	config.SetProviderService(new(UserProvider))
	// ------for hessian2------
	hessian.RegisterPOJO(&User{})

	cache = &UserDB{
		cacheMap:  make(map[string]*User, 16),
		cacheMapC: make(map[int64]*User, 16),
		lock:      sync.Mutex{},
	}

	cache.Add(&User{ID: "0001", Code: 1, Name: "tc", Age: 18, Time: time.Now()})
	cache.Add(&User{ID: "0002", Code: 2, Name: "ic", Age: 88, Time: time.Now()})
}

var cache *UserDB

// UserDB cache user.
type UserDB struct {
	// key is name, value is user obj
	cacheMap  map[string]*User
	cacheMapC map[int64]*User
	lock      sync.Mutex
}

// Add add user
func (db *UserDB) Add(u *User) bool {
	res := db.AddForName(u)
	if !res {
		return false
	}

	return db.AddForCode(u)
}

// AddForName add user with name
func (db *UserDB) AddForName(u *User) bool {
	db.lock.Lock()
	defer db.lock.Unlock()

	if len(u.Name) == 0 {
		return false
	}

	if _, ok := db.cacheMap[u.Name]; ok {
		return false
	}

	db.cacheMap[u.Name] = u
	return true
}

// AddForCode add user with code
func (db *UserDB) AddForCode(u *User) bool {
	db.lock.Lock()
	defer db.lock.Unlock()

	if u.Code == 0 {
		return false
	}

	if _, ok := db.cacheMapC[u.Code]; ok {
		return false
	}

	db.cacheMapC[u.Code] = u
	return true
}

// GetByName find user by name
func (db *UserDB) GetByName(n string) (*User, bool) {
	db.lock.Lock()
	defer db.lock.Unlock()

	r, ok := db.cacheMap[n]
	return r, ok
}

// GetByCode find user by code
func (db *UserDB) GetByCode(n int64) (*User, bool) {
	db.lock.Lock()
	defer db.lock.Unlock()

	r, ok := db.cacheMapC[n]
	return r, ok
}

// User user obj.
type User struct {
	ID   string    `json:"id,omitempty"`
	Code int64     `json:"code,omitempty"`
	Name string    `json:"name,omitempty"`
	Age  int32     `json:"age,omitempty"`
	Time time.Time `json:"time,omitempty"`
}

// UserProvider the dubbo provider.
// like: version: 1.0.0 group: test
type UserProvider struct{}

// CreateUser new user, PX config POST.
func (u *UserProvider) CreateUser(ctx context.Context, user *User) (*User, error) {
	println("Req CreateUser data:%#v", user)
	if user == nil {
		return nil, errors.New("not found")
	}
	_, ok := cache.GetByName(user.Name)
	if ok {
		return nil, errors.New("data is exist")
	}

	b := cache.Add(user)
	if b {
		return user, nil
	}

	return nil, errors.New("add error")
}

// GetUserByName query by name, single param, PX config GET.
func (u *UserProvider) GetUserByName(ctx context.Context, name string) (*User, error) {
	println("Req GetUserByName name:%#v", name)
	r, ok := cache.GetByName(name)
	if ok {
		println("Req GetUserByName result:%#v", r)
		return r, nil
	}
	return nil, nil
}

// GetUserByCode query by code, single param, PX config GET.
func (u *UserProvider) GetUserByCode(ctx context.Context, code int64) (*User, error) {
	println("Req GetUserByCode name:%#v", code)
	r, ok := cache.GetByCode(code)
	if ok {
		println("Req GetUserByCode result:%#v", r)
		return r, nil
	}
	return nil, nil
}

// GetUserTimeout query by name, will timeout for pixiu.
func (u *UserProvider) GetUserTimeout(ctx context.Context, name string) (*User, error) {
	println("Req GetUserTimeout name:%#v", name)
	// sleep 10s, pixiu config less than 10s.
	time.Sleep(10 * time.Second)
	r, ok := cache.GetByName(name)
	if ok {
		println("Req GetUserTimeout result:%#v", r)
		return r, nil
	}
	return nil, nil
}

// GetUserByNameAndAge query by name and age, two params, PX config GET.
func (u *UserProvider) GetUserByNameAndAge(ctx context.Context, name string, age int32) (*User, error) {
	println("Req GetUserByNameAndAge name:%s, age:%d", name, age)
	r, ok := cache.GetByName(name)
	if ok && r.Age == age {
		println("Req GetUserByNameAndAge result:%#v", r)
		return r, nil
	}
	return r, nil
}

// UpdateUser update by user struct, my be another struct, PX config POST or PUT.
func (u *UserProvider) UpdateUser(ctx context.Context, user *User) (bool, error) {
	println("Req UpdateUser data:%#v", user)
	r, ok := cache.GetByName(user.Name)
	if ok {
		if user.ID != "" {
			r.ID = user.ID
		}
		if user.Age >= 0 {
			r.Age = user.Age
		}
		return true, nil
	}
	return false, errors.New("not found")
}

// UpdateUserByName update by user struct, my be another struct, PX config POST or PUT.
func (u *UserProvider) UpdateUserByName(ctx context.Context, name string, user *User) (bool, error) {
	println("Req UpdateUserByName data:%#v", user)
	r, ok := cache.GetByName(name)
	if ok {
		if user.ID != "" {
			r.ID = user.ID
		}
		if user.Age >= 0 {
			r.Age = user.Age
		}
		return true, nil
	}
	return false, errors.New("not found")
}

// Reference return reference
func (u *UserProvider) Reference() string {
	return "UserProvider"
}

// JavaClassName return java class name
func (u User) JavaClassName() string {
	return "com.ikurento.user.User"
}

// nolint.
func println(format string, args ...interface{}) {
	fmt.Printf("\033[32;40m"+format+"\033[0m\n", args...)
}
