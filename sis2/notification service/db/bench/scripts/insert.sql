-- worker persists the notification in the inbox (write path step 6)
\set uid random(1, 200000)
INSERT INTO notifications (id, user_id, created_at, channel, title, body, status, is_read)
VALUES (nextval('notif_id_seq'), :uid, now(), :CH, 'Your order has shipped', 'Track your parcel in the app, delivery is expected tomorrow between 10:00 and 18:00', :ST0, false);
