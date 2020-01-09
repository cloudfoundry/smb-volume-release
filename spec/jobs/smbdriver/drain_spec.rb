require 'rspec'
require 'bosh/template/test'

describe 'smbdriver job' do
  let(:release) {Bosh::Template::Test::ReleaseDir.new(File.join(File.dirname(__FILE__), '../../..'))}
  let(:job) {release.job('smbdriver')}

  describe 'drain' do
    let(:template) {job.template('bin/drain')}

    context 'when configured with an admin port' do
      let(:manifest_properties) do
        {
            "admin_port" => "1111",
        }
      end

      it 'renders successfully' do
        tpl_output = template.render(manifest_properties)

        expect(tpl_output).not_to include("ADMIN_PORT=1111")
      end
    end
  end
end
