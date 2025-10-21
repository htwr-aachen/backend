# Run templ generation in watch mode
templ:
	templ generate --watch --proxy="http://localhost:8090" --open-browser=false -v

# Run air for Go hot reload
server:
	air run --disable-panikzettel

# Watch Tailwind CSS changes
tailwind:
	tailwindcss -i ./pkg/admin/assets/assets/css/input.css -o ./pkg/admin/assets/assets/css/output.css --watch

# Start development server with all watchers
dev:
	make -j3 tailwind templ server
