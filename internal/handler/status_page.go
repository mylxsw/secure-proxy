package handler

import (
	"fmt"
	"net/http"

	"github.com/mylxsw/secure-proxy/internal/template"
)

func (handler *AuthHandler) buildStatusHandler() func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html")

		userAuthInfo, err := handler.cookieManager.GetCookie(r)
		if err != nil {
			http.Redirect(rw, r, "/secure-proxy/auth", http.StatusSeeOther)
			return
		}

		if err := template.Render(statusPage, rw, userAuthInfo); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
			return
		}
	}
}

var statusPage = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Unified Authentication - Status</title>

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
	<div class="card card-default">
		<div class="card-header">Status</div>
		<div class="card-body text-left">
			<p>Current Login Information:</p>
			<li>User: <b>{{ .Account }}({{ .Name }})</b></li>
			<li>Login Time: <b>{{ .CreatedAt }}</b></li>
			<li>User Groups:
			{{- range $i, $group := .Groups }}
			<span class="badge rounded-pill bg-success">{{ $group }}</span>
			{{- end }}
			</li>
			<p class="mt-2"><a href="/secure-proxy/auth/logout" class="btn btn-primary">Logout</a></p>
		</div>
		<p class="mt-1 mb-3 text-muted">© 2022</p>
	</div>
  </main>

<script>

</script>
    
  </body>
</html>`
