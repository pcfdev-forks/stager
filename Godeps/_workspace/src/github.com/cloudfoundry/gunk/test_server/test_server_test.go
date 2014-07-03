package test_server_test

import (
	"bytes"
	. "github.com/cloudfoundry/gunk/test_server"
	"github.com/cloudfoundry/gunk/urljoiner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
)

var _ = Describe("TestServer", func() {
	var (
		resp *http.Response
		err  error
		s    *Server
	)

	BeforeEach(func() {
		s = New()
	})

	AfterEach(func() {
		s.Close()
	})

	Describe("allowing unhandled requests", func() {
		BeforeEach(func() {
			s.AllowUnhandledRequests = true
			s.UnhandledRequestStatusCode = http.StatusForbidden
			resp, err = http.Get(urljoiner.Join(s.URL(), "/foo"))
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should allow unhandled requests and respond with the passed in status code", func() {
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp.StatusCode).Should(Equal(http.StatusForbidden))

			data, err := ioutil.ReadAll(resp.Body)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(data).Should(BeEmpty())
		})

		It("should record the requests", func() {
			Ω(s.ReceivedRequestsCount()).Should(Equal(1))
		})
	})

	Describe("Request Handlers", func() {
		Describe("VerifyRequest", func() {
			BeforeEach(func() {
				s.Append(VerifyRequest("GET", "/foo"))
			})

			It("should verify the method, path", func() {
				resp, err = http.Get(urljoiner.Join(s.URL(), "/foo?baz=bar"))
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should also be possible to verify the rawQuery", func() {
				s.Set(0, VerifyRequest("GET", "/foo", "baz=bar"))
				resp, err = http.Get(urljoiner.Join(s.URL(), "/foo?baz=bar"))
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Describe("VerifyContentType", func() {
			BeforeEach(func() {
				s.Append(CombineHandlers(
					VerifyRequest("GET", "/foo"),
					VerifyContentType("application/octet-stream"),
				))
			})

			It("should verify the content type", func() {
				req, err := http.NewRequest("GET", urljoiner.Join(s.URL(), "/foo"), nil)
				Ω(err).ShouldNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/octet-stream")

				resp, err = http.DefaultClient.Do(req)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Describe("Verify BasicAuth", func() {
			BeforeEach(func() {
				s.Append(CombineHandlers(
					VerifyRequest("GET", "/foo"),
					VerifyBasicAuth("bob", "password"),
				))
			})

			It("should verify basic auth", func() {
				req, err := http.NewRequest("GET", urljoiner.Join(s.URL(), "/foo"), nil)
				Ω(err).ShouldNot(HaveOccurred())
				req.SetBasicAuth("bob", "password")

				resp, err = http.DefaultClient.Do(req)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Describe("VerifyJSON", func() {
			BeforeEach(func() {
				s.Append(CombineHandlers(
					VerifyRequest("POST", "/foo"),
					VerifyJSON(`{"a":3, "b":2}`),
				))
			})

			It("should verify the json body and the content type", func() {
				resp, err = http.Post(urljoiner.Join(s.URL(), "/foo"), "application/json", bytes.NewReader([]byte(`{"b":2, "a":3}`)))
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Describe("Respond", func() {
			BeforeEach(func() {
				s.Append(CombineHandlers(
					VerifyRequest("POST", "/foo"),
					Respond(http.StatusCreated, "sweet"),
				))
			})

			It("should return the response", func() {
				resp, err = http.Post(urljoiner.Join(s.URL(), "/foo"), "application/json", nil)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(resp.StatusCode).Should(Equal(http.StatusCreated))

				body, err := ioutil.ReadAll(resp.Body)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(body).Should(Equal([]byte("sweet")))
			})
		})

		Describe("ResponsePtr", func() {
			var code int
			var body string
			BeforeEach(func() {
				code = http.StatusOK
				body = "sweet"

				s.Append(CombineHandlers(
					VerifyRequest("POST", "/foo"),
					RespondPtr(&code, &body),
				))
			})

			It("should return the response", func() {
				code = http.StatusCreated
				body = "tasty"
				resp, err = http.Post(urljoiner.Join(s.URL(), "/foo"), "application/json", nil)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(resp.StatusCode).Should(Equal(http.StatusCreated))

				body, err := ioutil.ReadAll(resp.Body)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(body).Should(Equal([]byte("tasty")))
			})
		})
	})
})
