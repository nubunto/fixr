# Current Redis Architecture

**Keys**

 * **users :: Set**: holds the id's of registered users.
 * **users:%ID :: Map**: holds the info about the user, it's preferences, base currency, etc.

That's about it. Less is MOAR! 
