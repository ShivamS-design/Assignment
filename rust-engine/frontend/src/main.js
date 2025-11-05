// Simple frontend server
const express = require('express');
const path = require('path');

const app = express();
const port = process.env.PORT || 3000;

// Serve static files
app.use(express.static(path.join(__dirname, '..')));

// Serve index.html for all routes
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, '..', 'index.html'));
});

console.log(`🌐 WASM-as-OS Frontend starting on port ${port}`);
console.log(`   Access: http://localhost:${port}`);

app.listen(port, '0.0.0.0');