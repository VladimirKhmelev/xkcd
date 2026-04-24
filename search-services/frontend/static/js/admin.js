document.addEventListener('DOMContentLoaded', function () {
  const dropBtn = document.getElementById('drop-btn');
  if (dropBtn) {
    dropBtn.addEventListener('click', function (e) {
      if (!confirm('Are you sure you want to DROP the entire database? This cannot be undone.')) {
        e.preventDefault();
      }
    });
  }

  const alerts = document.querySelectorAll('.alert');
  alerts.forEach(function (el) {
    setTimeout(function () {
      el.style.transition = 'opacity 0.5s';
      el.style.opacity = '0';
      setTimeout(function () { el.remove(); }, 500);
    }, 4000);
  });
});
