package handler

import (
	"fmt"
	"net/http"

	"github.com/mylxsw/secure-proxy/internal/template"
)

func (handler *AuthHandler) buildIndexPageHandler() func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()

		values := make(map[string]string)
		values["auth"] = handler.conf.AuthType
		for _, k := range []string{"k0", "username", "password", "error"} {
			values[k] = r.Form.Get(k)
		}

		rw.Header().Add("Content-Type", "text/html")
		if err := template.Render(loginPage, rw, values); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
			return
		}
	}
}

var loginPage = `
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>统一身份认证 - 登录</title>

    <link href="/secure-proxy/assets/css/bootstrap.min.css" rel="stylesheet">

    <style>
      .bd-placeholder-img {
        font-size: 1.125rem;
        text-anchor: middle;
        -webkit-user-select: none;
        -moz-user-select: none;
        user-select: none;
      }

      @media (min-width: 768px) {
        .bd-placeholder-img-lg {
          font-size: 3.5rem;
        }
      }
    </style>

    <link href="/secure-proxy/assets/style.css?v=20220118" rel="stylesheet">
  </head>
  <body class="text-center">
    
<main class="form-signin">
  <form action="/secure-proxy/auth/login" method="POST" autocomplete="off">
    <h1 class="h3 mb-3 fw-normal">统一身份认证</h1>
    
    {{- if ne .error "" }}
    <div class="alert alert-danger" role="alert">{{ .error }}</div>
    {{- end }}

    {{- if eq .auth "ldap_local" }}
    <div class="form-floating">
      <select class="form-control" name="k0" id="k0">
        <option value="ldap" {{ if eq .k0 "ldap" }}selected{{ end }}>内部员工</option>
        <option value="local" {{ if eq .k0 "local" }}selected{{ end }}>外部用户</option>
      </select>
      <label for="k0">账号类型</label>
    </div>
    {{- end }}

    <div class="form-floating mt-3">
      <input type="text" class="form-control" id="floatingInput" name="username" value="{{ or .username "" }}" autocomplete="off" placeholder="">
      <label for="floatingInput">账号</label>
    </div>

    <div class="form-floating mt-3">
      <input type="password" class="form-control" id="floatingPassword" name="password" autocomplete="new-password">
      <label for="floatingPassword">密码</label>
    </div>

    <button class="w-100 btn btn-lg btn-primary" type="submit">登录</button>
    <p class="mt-5 mb-3 text-muted">&copy; 2022</p>
  </form>
</main>

<script>

</script>
    
  </body>
</html>
`
