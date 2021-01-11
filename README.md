# go_to_sheets
At this moment, this is quick and dirty. Intended for short lived, temporary situations.

This code will not give clear error messages. However, its still a handy utility.

In a nutshell, this runs a query in the config, and then dynamically builds the headers and inserts the data into a sheet. It overwrites everything, every time.

--

To build just checkout, and go get all the things you need, and then go build.

After that go over to google oauth, get yourself a client_credentials json file ( the web app one ), run this program from CLI.

It will give you a URL to visit in your browser, and then paste the code back into this.

Once done, you can run it over and over again.

If you are going to cron it, note: paths are relative. Mind the CWD. 
