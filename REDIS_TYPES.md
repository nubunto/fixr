# Current Redis Architecture

**Keys**

 * **users :: Set**: holds the id's of registered users.
 * **user:%ID :: Map**: holds the info about the user, it's preferences, base currency, etc.
 * **rates:%ID :: Set**: holds the preferred rates of the user.

That's about it. Less is MOAR! 
