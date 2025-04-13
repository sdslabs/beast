from flask import Flask
from flask_sqlalchemy import SQLAlchemy
from flask_login import LoginManager
from flask_wtf.csrf import CSRFProtect
import os
from apscheduler.schedulers.background import BackgroundScheduler

db = SQLAlchemy()
login_manager = LoginManager()
csrf = CSRFProtect()
scheduler = BackgroundScheduler()

def clear_database():
    from app.models import User, Note
    with scheduler.app.app_context():
        db.session.query(Note).delete()
        db.session.query(User).delete()
        db.session.commit()

def create_app():
    app = Flask(__name__)

    app.config['SECRET_KEY'] = os.environ.get('SECRET_KEY', 'default_secret_key')
    app.config['SQLALCHEMY_DATABASE_URI'] = f"postgresql://{os.environ.get('POSTGRES_USER')}:{os.environ.get('POSTGRES_PASSWORD')}@{os.environ.get('POSTGRES_HOST')}:{os.environ.get('POSTGRES_PORT', 5432)}/{os.environ.get('POSTGRES_DB')}"

    db.init_app(app)
    login_manager.init_app(app)
    csrf.init_app(app)

    from app.models import User, Note

    @login_manager.user_loader
    def load_user(user_id):
        return User.query.get(int(user_id))

    from app.views import main as main_blueprint
    app.register_blueprint(main_blueprint)

    with app.app_context():
        db.create_all()

    scheduler.add_job(func=clear_database, trigger="interval", minutes=60)
    scheduler.start()

    scheduler.app = app

    return app
