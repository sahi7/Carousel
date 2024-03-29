# Generated by Django 4.0.2 on 2022-02-28 11:38

from django.conf import settings
from django.db import migrations, models
import django.db.models.deletion


class Migration(migrations.Migration):

    dependencies = [
        ('restau', '0001_initial'),
    ]

    operations = [
        migrations.AddField(
            model_name='order',
            name='created_by',
            field=models.OneToOneField(blank=True, null=True, on_delete=django.db.models.deletion.CASCADE, to=settings.AUTH_USER_MODEL),
        ),
        migrations.AddField(
            model_name='restaurant',
            name='owner',
            field=models.ForeignKey(null=True, on_delete=django.db.models.deletion.CASCADE, to='restau.owner'),
        ),
    ]
