# hnjobs
hnjobs is a private [Hacker News](https://news.ycombinator.com/) ["Who is hiring?"](https://news.ycombinator.com/user?id=whoishiring) thread filter and search tool. It lists all of the posts on a "Who is hiring?" thread and provides buttons to filter for "Remote", "Interns" and "Visa" jobs, an input box for ad-hoc (case-insensitive) filtering of posts using [Javascript regular expressions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Regular_expressions), the ability to remove posts from the post listing, and the ability to restore all removed posts.

## Deployed instance
A deployment of the tool is live at https://nchelluri.github.io/hnjobs/ and is updated once an hour for the first ten days of each month.

## Running locally
You can run the program locally with the command `go run .` which will generate a file called `index.html` that you can view in your web browser.

## Privacy
All data is stored locally using [localStorage](https://developer.mozilla.org/en-US/docs/Web/API/Window/localStorage). No data about any filters you use, links you click, posts you remove from the listing, etc., is ever sent anywhere. There are no analytics.
