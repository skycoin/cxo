cleanup
=======

The CXO keeps all Root objects and all related objects. And end-user
have to remove them manually. Removing is not trivial. A Root object
can't be removed if it's last. Actually, it can be removed, but next
Root objects can not be created for some reasons. The CXO uses lazy
strategy everywhere and objects are not loaded to memory since they
are not needed. And if you remvoe Root 1, then creating of Root 2
(based on the Root 1) fails if some objects of the Root 1 was removed
from DB.

More then that a Root objects should be replicated by peer. And this
requires some time.

The recommended way is removing a Root after creating one. Keepping
some "safety cushion". May be. Ha-ha.

There is [`cxoutils/`](../../cxoutils) package that used in the example.

The `cleanup` example generates new Root objects every second and removes
Root and related objects every 30s keeping last 5 Root objects. And printing
statistic of the objects.

