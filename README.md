# olympus

Terraform, GitOps, and Observability

## Overview

This is currently an experimental project in Terraform and GitOps.

I have seen a number of other systems and they all fall short of meeting my needs in some way. Some
of them are heavily committed to their paths.

This is an experimental project to find out how to address those shortcoming and explore needs. It
may eventually be a full blown open source project intented as an alternative.

## Initial Milestone

The first milestone is collecting the results of "terraform plan" from multiple environments and
displaying them in a useful manner.

First class considerations:

* There are many clusters. We want views across clusters
* Security -- specificly the ownership and control of secret information
* Flexibility -- we are not going to assume how to run the various tools. Tools should be extensible
  and replacable.