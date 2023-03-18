/*
 * Copyright (C) Max Romanov
 * Copyright (C) NGINX, Inc.
 */

package unit

/*
#include "nxt_cgo_lib.h"
*/
import "C"

import (
	"net/http"
	"time"
)

type response struct {
	header     http.Header
	header_sent bool
	c_req      *C.nxt_unit_request_info_t
	ch         chan int
}

func (r *response) Header() http.Header {
	return r.header
}

func (r *response) Write(p []byte) (n int, err error) {
	if !r.header_sent {
		r.WriteHeader(http.StatusOK)
	}

	l := len(p)
	written := int(0)
	br := buf_ref(p)

	for written < l {
		res := C.nxt_cgo_response_write(r.c_req, br, C.uint32_t(l - written))

		written += int(res)
		br += C.uintptr_t(res)

		if (written < l) {
			if r.ch == nil {
				r.ch = make(chan int, 2)
			}

			wait_shm_ack(r.ch)
		}
	}

	return written, nil
}

func (r *response) WriteHeader(code int) {
	if r.header_sent {
		nxt_go_warn("multiple response.WriteHeader calls")
		return
	}
	r.header_sent = true

	// Set a default Content-Type
	if _, hasType := r.header["Content-Type"]; !hasType {
		r.header.Add("Content-Type", "text/html; charset=utf-8")
	}

	fields := 0
	fields_size := 0

	for k, vv := range r.header {
		for _, v := range vv {
			fields++
			fields_size += len(k) + len(v)
		}
	}

	C.nxt_unit_response_init(r.c_req, C.uint16_t(code), C.uint32_t(fields),
		C.uint32_t(fields_size))

	for k, vv := range r.header {
		for _, v := range vv {
			C.nxt_unit_response_add_field(r.c_req, str_ref(k), C.uint8_t(len(k)),
				str_ref(v), C.uint32_t(len(v)))
		}
	}

	C.nxt_unit_response_send(r.c_req)
}

func (r *response) Flush() {
	if !r.header_sent {
		r.WriteHeader(http.StatusOK)
	}
}

var observer_registry_ observable

func wait_shm_ack(c chan int) {
	observer_registry_.attach(c)

	_ = <-c
}

//export nxt_go_shm_ack_handler
func nxt_go_shm_ack_handler(ctx *C.nxt_unit_ctx_t) {
	observer_registry_.notify(1)
}



type CacheEntry struct {
    Response *response
    Expiry   time.Time
}

type Cache struct {
    cache map[string]*CacheEntry
}

func NewCache() *Cache {
    return &Cache{
        cache: make(map[string]*CacheEntry),
    }
}

func (c *Cache) Get(key string) *CacheEntry {
    entry, ok := c.cache[key]
    if !ok {
        return nil
    }
    if entry.Expiry.Before(time.Now()) {
        delete(c.cache, key)
        return nil
    }
    return entry
}

func (c *Cache) Set(key string, entry *CacheEntry) {
    c.cache[key] = entry
}

func (r *response) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    var cacheKey string
    if req.Method == http.MethodGet {
        cacheKey = req.URL.Path
        cacheEntry := cache.Get(cacheKey)
        if cacheEntry != nil {
            r = cacheEntry.Response
        }
    }
    r.ResponseWriter = w
    r.Request = req
    r.writeHeader = true
    handler(r, req)
    if req.Method == http.MethodGet && r.statusCode == http.StatusOK {
        cache.Set(cacheKey, &CacheEntry{
            Response: r,
            Expiry:   time.Now().Add(5 * time.Minute), // Cache for 5 minutes
        })
    }
}

type ConnectionPool struct {
    queue chan *nxt_unit_request_info_t
}

func NewConnectionPool(size int) *ConnectionPool {
    return &ConnectionPool{
        queue: make(chan *nxt_unit_request_info_t, size),
    }
}

func (p *ConnectionPool) Get() *nxt_unit_request_info_t {
    select {
    case req := <-p.queue:
        return req
    default:
        return nil
    }
}

func (p *ConnectionPool) Put(req *nxt_unit_request_info_t) {
    select {
    case p.queue <- req:
    default:
        nxt_unit_response_done(req, NXT_UNIT_OK)
    }
}

var connectionPool = NewConnectionPool(10)

func handler(r *response, req *http.Request) {
    c_req := connectionPool.Get()
    if c_req == nil {
        nxt_unit_request_info_alloc(r.c_req)
        c_req = r.c_req
    }

    defer connectionPool.Put(c_req)
}


var cReqPool = sync.Pool{
    New: func() interface{} {
        return new(nxt_unit_request_info_t)
    },
}

func handler(r *response, req *http.Request) {
    cReq := cReqPool.Get().(*nxt_unit_request_info_t)
    defer cReqPool.Put(cReq)

    nxt_unit_request_info_alloc(cReq)
    // rest of the code
}