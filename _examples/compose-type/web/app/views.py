import os
from flask import Blueprint, render_template, redirect, url_for, request, jsonify,make_response,current_app
from flask_login import login_user, login_required, logout_user, current_user
from urllib.parse import urlparse, urljoin
from app import db
from app.models import User, Note
from app.forms import LoginForm, RegisterForm, NoteForm, ContactForm, ReportForm
import hashlib
import time
import bleach
import datetime
import requests
import subprocess
import threading
from markupsafe import escape
from html import escape as html_escape


main = Blueprint('main', __name__)

BASE_URL = os.getenv('BASE_URL', 'http://0.0.0.0')
BOT_URL = os.getenv('BOT_URL', 'http://0.0.0.0:8000')

reporting_users = set()
reporting_lock = threading.Lock()

def custom_escape(text):
    if text is None:
        return ''
    return text.replace('$', '&#36;').replace('{', '&#123;').replace('}', '&#125;').replace('[', '&#91;').replace(']', '&#93;')

@main.errorhandler(500)
def internal_server_error(error):
    current_app.logger.error(f"Server Error: {error}, Path: {request.path}")

    return render_template('500.html'), 500


@main.errorhandler(401)
def unauthorized_error(error):
    return render_template('401.html'), 401

def is_secure_link(target): 
    test_url = urlparse(urljoin(request.host_url, target))
    if target=="/":
        return False
    if test_url.scheme in ('http', 'https'):
        return True
    return False

@main.route('/')
def index():
    if current_user.is_authenticated:
        return redirect(url_for('main.home'))  
    else:
        return redirect(url_for('main.login'))  

@main.route('/home')
def home():
    return render_template('home.html')

@main.route('/api/notes/fetch/<note_id>', methods=['GET'])
def fetch(note_id):
    if len(note_id) != 32 or not all(c in '0123456789abcdef' for c in note_id):
        return jsonify({'error': 'Invalid note ID format'}), 400

    note = Note.query.get(note_id)
    if note:
        return jsonify({'content': note.content, 'note_id': note.id})
    return jsonify({'error': 'Note not found'}), 404

@main.route('/api/notes/store', methods=['POST'])
@login_required
def store():
    data = request.get_json()
    content = data.get('content')

    if not content:
        return jsonify(success=False, error="Note content cannot be empty"), 400
    
    sanitized_content = bleach.clean(content)

    new_note = Note(user_id=current_user.id, content=sanitized_content)
    db.session.add(new_note)
    db.session.commit()

    return jsonify({'success': 'Note stored', 'note_id': new_note.id})

@main.route('/register', methods=['GET', 'POST'])
def register():
    form = RegisterForm()
    error_register = ''
    if form.validate_on_submit():
        user = User.query.filter_by(username=form.username.data).first()
        if user:
            error_register='Username already exists. Please choose a different one.'
        else:
            user = User(username=form.username.data, password=form.password.data)
            db.session.add(user)
            db.session.commit()
            login_user(user)
            return redirect(url_for('main.home'))
    elif request.method == 'POST':
        error_register='Registration Unsuccessful. Please check the errors and try again.'
    return render_template('register.html', form=form)

@main.route('/login', methods=['GET', 'POST'])
def login():
    error_login = ''
    form = LoginForm()
    if form.validate_on_submit():
        user = User.query.filter_by(username=form.username.data).first()
        if user and user.password == form.password.data:
            login_user(user)
            return redirect(url_for('main.home'))
        else:
            error_login='Login Unsuccessful. Please check username and password'
    return render_template('login.html', form=form,error_login=error_login)

@main.route('/create', methods=['GET', 'POST'])
@login_required
def create_note():
    form = NoteForm()
    if form.validate_on_submit():
        note = Note(user_id=current_user.id, content=form.content.data)
        db.session.merge(note)
        db.session.commit()
        return redirect(url_for('main.view_note', note=note.id))
    return render_template('create.html', form=form)

@main.route('/view', methods=['GET'])
def view_note():
    note_id = request.args.get('note') or ''  
    name_param = request.args.get('name')
    safe_name = custom_escape(name_param)    
    random_hash = hashlib.md5(str(time.time()).encode()).hexdigest()
    name = request.args.get('name') if safe_name else (current_user.username if current_user.is_authenticated else "Guest")
    again_name_safe=custom_escape(name)
    return render_template('view.html', note_id=note_id, username=again_name_safe,random_hash=random_hash)

@main.route('/iframe_content', methods=['GET'])
def iframe_content():
    return render_template('iframe_content.html')


@main.route('/bay', methods=['GET', 'POST'])
def bay():
    return_url = request.args.get('return', '/')    
    form = ContactForm() 
    if request.method == 'POST':
        if form.validate_on_submit():
            if is_secure_link(return_url):
                return redirect(return_url)
            return redirect(url_for('main.home'))
    else:
        if is_secure_link(return_url):
            return redirect(return_url)
    return render_template('bay.html', form=form, return_url=return_url)

@main.route('/logout')
@login_required
def logout():
    logout_user()
    return redirect(url_for('main.home'))

@main.route('/report', methods=['GET', 'POST'])
@login_required
def report():
    form = ReportForm()
    log=''
    if form.validate_on_submit():
        note_url = form.note_url.data
        parsed_url = urlparse(note_url)
        base_url_parsed = urlparse(BASE_URL)
        if not parsed_url.scheme.startswith('http'):
            log='URL must begin with http(s)://'
    
        elif parsed_url.netloc == base_url_parsed.netloc and parsed_url.path == '/view' and 'note=' in parsed_url.query:
            note_id = parsed_url.query.split('=')[-1]

            try:
                if len(note_id) == 32 and all(c in '0123456789abcdef' for c in note_id):
                    with reporting_lock:
                        if current_user.id in reporting_users:
                            log='You already have a report in progress. Please respect our moderation capabilities.'
                        else:
                            reporting_users.add(current_user.id)
                            threading.Thread(target=bot, args=(
                                note_url, current_user.id)).start()
                            log='Note reported successfully'
            except ValueError:
                pass
        else:
            log=f'Please provide a valid note URL eg. http://{base_url_parsed.netloc}/view?note=7c1ce3ded0e8b97db1bdd2fabba701b4'

    return render_template('report.html', form=form,messages=log)
def bot(note_url, user_id):
    try:
        response = requests.post(f"{BOT_URL}", data={"url": note_url})
        if response.status_code == 200:
            pass
        else:
            pass
    finally:
        with reporting_lock:
            reporting_users.remove(user_id)