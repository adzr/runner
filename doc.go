/*
Copyright 2018 Ahmed Zaher

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package runner provides an interface to define the lifecycle of service like components and a function to manage them.

Brief

This library provides a way to manage the lifecycle of service like components.

Usage

	$ go get -u bitbucket.org/azaher/runner

Then, import the package:

  import (
    "bitbucket.org/azaher/runner"
  )

Finally, just implement the Runnable interface and pass it to the Run() function.

*/
package runner
