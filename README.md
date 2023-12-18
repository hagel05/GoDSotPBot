Initial commit, lack of formatting.

Currently this sets up the infrastructure to deploy an AWS lambda function and a simple hello world go project.

Thing still needed to get this to do what is needed.  
1. API gateway terraform
2. Go code to recieve slashcommands and provide output based on that input
3. A design decision on how I will store and read data for a simple list to be modified year over year

Another item to take care of is to move and organize the current terraform code.  The terraform directory exists already, I just need to do the work and test the changes.