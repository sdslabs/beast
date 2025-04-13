from app import db
from flask_login import UserMixin
import hashlib
import time

class User(db.Model, UserMixin):
    id = db.Column(db.Integer, primary_key=True)
    username = db.Column(db.String(150), unique=True, nullable=False)
    password = db.Column(db.String(150), nullable=False)

class Note(db.Model):
    id = db.Column(db.String(32), primary_key=True, default=lambda: generate_md5_id())
    user_id = db.Column(db.Integer, db.ForeignKey('user.id'), nullable=False)
    content = db.Column(db.Text, nullable=False)
    user = db.relationship('User', backref=db.backref('notes', lazy=True))

def generate_md5_id():
    # Use current time (in seconds) to generate a unique MD5 hash.
    current_time = str(time.time()).encode('utf-8')
    return hashlib.md5(current_time).hexdigest()  # Generates a 32-character MD5 hash
