# Generated by Django 4.0.2 on 2022-03-04 05:43

from django.db import migrations


class Migration(migrations.Migration):

    dependencies = [
        ('restau', '0004_rename_account_status_user_user_status_and_more'),
    ]

    operations = [
        migrations.RenameField(
            model_name='user',
            old_name='Qualification',
            new_name='qualification',
        ),
    ]