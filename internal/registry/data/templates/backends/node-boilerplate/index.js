const http = require('http')

const handler = (req, res) => {
  // valla:cors
  if (req.method === 'OPTIONS') {
    res.writeHead(204)
    res.end()
    return
  }
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify({ status: 'ok' }))
    return
  }
  res.writeHead(404)
  res.end()
}

const port = process.env.PORT || 3000
http.createServer(handler).listen(port, () => {
  console.log(`Server running on :${port}`)
})
